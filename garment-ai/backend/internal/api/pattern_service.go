package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"garment-ai/backend/internal/config"
)

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type patternPiece struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Closed           bool     `json:"closed"`
	Points           []point  `json:"points"`
	SeamAllowanceMM  float64  `json:"seamAllowanceMm"`
	GrainLine        string   `json:"grainLine"`
	Notches          []point  `json:"notches,omitempty"`
	DrillHoles       []point  `json:"drillHoles,omitempty"`
	ConstructionNote string   `json:"constructionNote,omitempty"`
	GradeRule        string   `json:"gradeRule,omitempty"`
}

type patternMetadata struct {
	FileID      string         `json:"fileId"`
	ObjectKey   string         `json:"objectKey"`
	PatternJSON map[string]any `json:"patternJson"`
	FileSizeKB  int            `json:"fileSizeKb"`
	CacheHit    bool           `json:"cacheHit"`
	ExporterVersion string     `json:"exporterVersion"`
}

const dxfExporterVersion = "stage6-dxf-v4"

func generateIndustrialPattern(ctx context.Context, cfg config.Config, params map[string]any) (patternMetadata, error) {
	var empty patternMetadata

	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return empty, fmt.Errorf("create minio client: %w", err)
	}

	if err := ensureBucket(ctx, client, cfg.MinioBucket); err != nil {
		return empty, err
	}

	normalized := normalizePatternParams(params)
	hashInput := make(map[string]any, len(normalized)+1)
	for key, value := range normalized {
		hashInput[key] = value
	}
	hashInput["_dxfExporterVersion"] = dxfExporterVersion
	digestBytes := sha256.Sum256(mustJSON(hashInput))
	fileID := fmt.Sprintf("%x", digestBytes)
	dxfObjectKey := fmt.Sprintf("patterns/generated/%s.dxf", fileID)
	metadataObjectKey := fmt.Sprintf("patterns/generated/%s.json", fileID)

	if metadata, err := readPatternMetadata(ctx, client, cfg.MinioBucket, metadataObjectKey); err == nil {
		if metadata.ExporterVersion == dxfExporterVersion {
			if _, err := client.StatObject(ctx, cfg.MinioBucket, dxfObjectKey, minio.StatObjectOptions{}); err == nil {
			metadata.CacheHit = true
			return metadata, nil
			}
		}
	}

	patternJSON := buildPatternJSON(normalized)
	dxfContent := buildDXFDocument(patternJSON)

	if _, err := client.PutObject(
		ctx,
		cfg.MinioBucket,
		dxfObjectKey,
		bytes.NewReader(dxfContent),
		int64(len(dxfContent)),
		minio.PutObjectOptions{ContentType: "application/dxf"},
	); err != nil {
		return empty, fmt.Errorf("upload dxf: %w", err)
	}

	metadata := patternMetadata{
		FileID:      fileID,
		ObjectKey:   dxfObjectKey,
		PatternJSON: patternJSON,
		FileSizeKB:  maxInt(1, int(math.Ceil(float64(len(dxfContent))/1024.0))),
		CacheHit:    false,
		ExporterVersion: dxfExporterVersion,
	}
	metadataPayload, err := json.Marshal(metadata)
	if err != nil {
		return empty, fmt.Errorf("marshal pattern metadata: %w", err)
	}

	if _, err := client.PutObject(
		ctx,
		cfg.MinioBucket,
		metadataObjectKey,
		bytes.NewReader(metadataPayload),
		int64(len(metadataPayload)),
		minio.PutObjectOptions{ContentType: "application/json"},
	); err != nil {
		return empty, fmt.Errorf("upload pattern metadata: %w", err)
	}

	return metadata, nil
}

func readPatternMetadata(ctx context.Context, client *minio.Client, bucket, objectKey string) (patternMetadata, error) {
	var metadata patternMetadata

	object, err := client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return metadata, err
	}
	defer object.Close()

	stat, err := object.Stat()
	if err != nil {
		return metadata, err
	}
	if stat.Size == 0 {
		return metadata, fmt.Errorf("empty metadata")
	}

	if err := json.NewDecoder(object).Decode(&metadata); err != nil {
		return metadata, err
	}

	return metadata, nil
}

func normalizePatternParams(params map[string]any) map[string]any {
	normalized := sampleGarmentParams()
	for key, value := range params {
		normalized[key] = value
	}

	for _, key := range []string{
		"length", "chest", "shoulder", "neck", "sleeve_length", "cuff", "waist", "hip",
		"collar_width", "collar_depth", "placket_length", "pocket_width", "pocket_height",
		"pocket_position_x", "pocket_position_y", "dart_length", "dart_width", "seam_allowance",
	} {
		normalized[key] = numericValue(normalized[key], numericValue(sampleGarmentParams()[key], 0))
	}

	for _, key := range []string{"button_count", "pocket_count", "dart_count"} {
		normalized[key] = int(numericValue(normalized[key], numericValue(sampleGarmentParams()[key], 0)))
	}

	for _, key := range []string{"garment_type", "silhouette", "collar_type", "sleeve_type", "sleeve_cuff_type", "placket_type", "pocket_type", "dart_position", "notch_type", "grain_line", "grade_rule"} {
		normalized[key] = textValue(normalized[key], fmt.Sprint(sampleGarmentParams()[key]))
	}

	return normalized
}

func numericValue(raw any, fallback float64) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		parsed, err := value.Float64()
		if err == nil {
			return parsed
		}
	case string:
		var parsed float64
		_, err := fmt.Sscanf(strings.TrimSpace(value), "%f", &parsed)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func textValue(raw any, fallback string) string {
	if raw == nil {
		return fallback
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return fallback
	}
	return value
}

func buildPatternJSON(params map[string]any) map[string]any {
	lengthMM := numericValue(params["length"], 65) * 10.0
	chestHalfMM := numericValue(params["chest"], 96) * 5.0
	waistHalfMM := numericValue(params["waist"], 84) * 5.0
	hipHalfMM := numericValue(params["hip"], 98) * 5.0
	shoulderHalfMM := numericValue(params["shoulder"], 42) * 5.0
	sleeveLengthMM := numericValue(params["sleeve_length"], 60) * 10.0
	cuffHalfMM := numericValue(params["cuff"], 22) * 5.0
	collarWidthMM := numericValue(params["collar_width"], 8) * 10.0
	collarDepthMM := numericValue(params["collar_depth"], 5) * 10.0
	pocketWidthMM := numericValue(params["pocket_width"], 12) * 10.0
	pocketHeightMM := numericValue(params["pocket_height"], 14) * 10.0
	seamAllowanceMM := numericValue(params["seam_allowance"], 1.0) * 10.0
	grainLine := textValue(params["grain_line"], "经向")
	gradeRule := textValue(params["grade_rule"], "国标女")
	notchType := textValue(params["notch_type"], "标准剪口")

	front := patternPiece{
		ID:              "front",
		Name:            "前片",
		Category:        "body",
		Closed:          true,
		SeamAllowanceMM: seamAllowanceMM,
		GrainLine:       grainLine,
		GradeRule:       gradeRule,
		Points: []point{
			{X: 0, Y: 0},
			{X: shoulderHalfMM, Y: 0},
			{X: chestHalfMM, Y: lengthMM * 0.28},
			{X: waistHalfMM, Y: lengthMM * 0.62},
			{X: hipHalfMM, Y: lengthMM},
			{X: 0, Y: lengthMM},
		},
		Notches:          []point{{X: chestHalfMM * 0.94, Y: lengthMM * 0.22}},
		DrillHoles:       []point{{X: waistHalfMM * 0.58, Y: lengthMM * 0.56}},
		ConstructionNote: notchType,
	}

	backOffset := hipHalfMM + 180.0
	back := patternPiece{
		ID:              "back",
		Name:            "后片",
		Category:        "body",
		Closed:          true,
		SeamAllowanceMM: seamAllowanceMM,
		GrainLine:       grainLine,
		GradeRule:       gradeRule,
		Points: []point{
			{X: backOffset, Y: 0},
			{X: backOffset + shoulderHalfMM, Y: 0},
			{X: backOffset + chestHalfMM, Y: lengthMM * 0.24},
			{X: backOffset + waistHalfMM, Y: lengthMM * 0.6},
			{X: backOffset + hipHalfMM, Y: lengthMM},
			{X: backOffset, Y: lengthMM},
		},
		Notches:          []point{{X: backOffset + chestHalfMM * 0.96, Y: lengthMM * 0.19}},
		ConstructionNote: notchType,
	}

	sleeveOffsetY := lengthMM + 180.0
	sleeve := patternPiece{
		ID:              "sleeve",
		Name:            textValue(params["sleeve_type"], "长袖"),
		Category:        "sleeve",
		Closed:          true,
		SeamAllowanceMM: seamAllowanceMM,
		GrainLine:       grainLine,
		GradeRule:       gradeRule,
		Points: []point{
			{X: 0, Y: sleeveOffsetY},
			{X: chestHalfMM * 0.8, Y: sleeveOffsetY},
			{X: chestHalfMM * 0.92, Y: sleeveOffsetY + sleeveLengthMM * 0.18},
			{X: cuffHalfMM, Y: sleeveOffsetY + sleeveLengthMM},
			{X: 0, Y: sleeveOffsetY + sleeveLengthMM},
		},
		Notches:          []point{{X: chestHalfMM * 0.45, Y: sleeveOffsetY + sleeveLengthMM * 0.08}},
		ConstructionNote: textValue(params["sleeve_cuff_type"], "单扣"),
	}

	collarOffset := chestHalfMM + backOffset + 180.0
	collar := patternPiece{
		ID:              "collar",
		Name:            textValue(params["collar_type"], "翻领"),
		Category:        "collar",
		Closed:          true,
		SeamAllowanceMM: seamAllowanceMM,
		GrainLine:       grainLine,
		GradeRule:       gradeRule,
		Points: []point{
			{X: collarOffset, Y: 0},
			{X: collarOffset + collarWidthMM * 2.2, Y: 0},
			{X: collarOffset + collarWidthMM * 2.0, Y: collarDepthMM},
			{X: collarOffset, Y: collarDepthMM},
		},
		ConstructionNote: textValue(params["collar_type"], "翻领"),
	}

	pieces := []patternPiece{front, back, sleeve, collar}

	if textValue(params["placket_type"], "单排扣") != "暗门襟" {
		pieces = append(pieces, patternPiece{
			ID:              "placket",
			Name:            textValue(params["placket_type"], "单排扣"),
			Category:        "placket",
			Closed:          true,
			SeamAllowanceMM: seamAllowanceMM,
			GrainLine:       grainLine,
			GradeRule:       gradeRule,
			Points: []point{
				{X: collarOffset, Y: collarDepthMM + 80},
				{X: collarOffset + 45, Y: collarDepthMM + 80},
				{X: collarOffset + 45, Y: collarDepthMM + 80 + numericValue(params["placket_length"], 20)*10},
				{X: collarOffset, Y: collarDepthMM + 80 + numericValue(params["placket_length"], 20)*10},
			},
			ConstructionNote: fmt.Sprintf("button-count:%d", int(numericValue(params["button_count"], 6))),
		})
	}

	pocketCount := int(numericValue(params["pocket_count"], 2))
	if textValue(params["pocket_type"], "贴袋") != "无" && pocketCount > 0 {
		for index := 0; index < pocketCount; index++ {
			offsetX := float64(index) * (pocketWidthMM + 35)
			pieces = append(pieces, patternPiece{
				ID:              fmt.Sprintf("pocket-%d", index+1),
				Name:            fmt.Sprintf("口袋-%d", index+1),
				Category:        "pocket",
				Closed:          true,
				SeamAllowanceMM: seamAllowanceMM,
				GrainLine:       grainLine,
				GradeRule:       gradeRule,
				Points: []point{
					{X: collarOffset + offsetX, Y: lengthMM + 210},
					{X: collarOffset + offsetX + pocketWidthMM, Y: lengthMM + 210},
					{X: collarOffset + offsetX + pocketWidthMM, Y: lengthMM + 210 + pocketHeightMM},
					{X: collarOffset + offsetX, Y: lengthMM + 210 + pocketHeightMM},
				},
				ConstructionNote: textValue(params["pocket_type"], "贴袋"),
			})
		}
	}

	totalArea := 0.0
	for _, piece := range pieces {
		totalArea += polygonArea(piece.Points)
	}

	pieceMaps := make([]map[string]any, 0, len(pieces))
	for _, piece := range pieces {
		pieceMaps = append(pieceMaps, map[string]any{
			"id":               piece.ID,
			"name":             piece.Name,
			"category":         piece.Category,
			"closed":           piece.Closed,
			"points":           piece.Points,
			"seamAllowanceMm":  piece.SeamAllowanceMM,
			"grainLine":        piece.GrainLine,
			"notches":          piece.Notches,
			"drillHoles":       piece.DrillHoles,
			"constructionNote": piece.ConstructionNote,
			"gradeRule":        piece.GradeRule,
		})
	}

	return map[string]any{
		"version":    "stage6-v1",
		"units":      "mm",
		"garmentType": textValue(params["garment_type"], "shirt"),
		"silhouette": textValue(params["silhouette"], "H型"),
		"metadata": map[string]any{
			"pieceCount":       len(pieceMaps),
			"estimatedAreaMm2": math.Round(totalArea),
			"gradeRule":        gradeRule,
			"notchType":        notchType,
			"seamAllowanceMm":  seamAllowanceMM,
		},
		"pieces": pieceMaps,
	}
}

func buildDXFDocument(patternJSON map[string]any) []byte {
	var builder strings.Builder
	handleSeed := 0x200

	pieces, _ := patternJSON["pieces"].([]map[string]any)
	if pieces == nil {
		if rawPieces, ok := patternJSON["pieces"].([]any); ok {
			pieces = make([]map[string]any, 0, len(rawPieces))
			for _, raw := range rawPieces {
				if pieceMap, ok := raw.(map[string]any); ok {
					pieces = append(pieces, pieceMap)
				}
			}
		}
	}

	minX, minY, maxX, maxY := drawingBounds(pieces)
	writeDXFHeader(&builder, minX, minY, maxX, maxY)
	writeDXFTables(&builder, pieces)
	builder.WriteString("0\nSECTION\n2\nBLOCKS\n0\nENDSEC\n")
	builder.WriteString("0\nSECTION\n2\nENTITIES\n")

	for _, piece := range pieces {
		pieceID := textValue(piece["id"], "piece")
		pieceName := textValue(piece["name"], pieceID)
		layerName := dxfLayerName(pieceID)
		points := decodePoints(piece["points"])
		if len(points) == 0 {
			continue
		}

		writePolylineEntity(&builder, &handleSeed, layerName, points, true)
		seamPoints := expandPolygon(points, numericValue(piece["seamAllowanceMm"], 10))
		writePolylineEntity(&builder, &handleSeed, "SEAM_ALLOWANCE", seamPoints, true)

		anchor := centroid(points)
		writeTextEntity(&builder, &handleSeed, "ANNOTATION", asciiLabel(pieceID, pieceName), anchor.X, anchor.Y)
		writeTextEntity(&builder, &handleSeed, "ANNOTATION", fmt.Sprintf("CAT %s", asciiToken(textValue(piece["category"], "body"))), anchor.X, anchor.Y+16)
		writeTextEntity(&builder, &handleSeed, "ANNOTATION", fmt.Sprintf("SA %.1fmm", numericValue(piece["seamAllowanceMm"], 10)), anchor.X, anchor.Y+32)
		writeTextEntity(&builder, &handleSeed, "ANNOTATION", fmt.Sprintf("GL %s", asciiToken(textValue(piece["grainLine"], "warp"))), anchor.X, anchor.Y+48)
		if note := asciiToken(textValue(piece["constructionNote"], "")); note != "" {
			writeTextEntity(&builder, &handleSeed, "ANNOTATION", fmt.Sprintf("NOTE %s", note), anchor.X, anchor.Y+64)
		}

		grainStart, grainEnd := grainLine(points)
		writeLineEntity(&builder, &handleSeed, "GRAIN_LINE", grainStart, grainEnd)
		writeArrowEntity(&builder, &handleSeed, "GRAIN_LINE", grainStart, grainEnd)
		writeArrowEntity(&builder, &handleSeed, "GRAIN_LINE", grainEnd, grainStart)

		for _, vertex := range points {
			writePointEntity(&builder, &handleSeed, layerName+"_PTS", vertex)
		}

		for _, notch := range decodePoints(piece["notches"]) {
			writeCircleEntity(&builder, &handleSeed, "NOTCH", notch, 3)
			writeCrossEntity(&builder, &handleSeed, "NOTCH", notch, 6)
		}
		for _, hole := range decodePoints(piece["drillHoles"]) {
			writeCircleEntity(&builder, &handleSeed, "DRILL", hole, 2)
			writePointEntity(&builder, &handleSeed, "DRILL", hole)
		}
	}

	builder.WriteString("0\nENDSEC\n")
	builder.WriteString("0\nSECTION\n2\nOBJECTS\n0\nENDSEC\n0\nEOF\n")
	return []byte(builder.String())
}

func writeDXFHeader(builder *strings.Builder, minX, minY, maxX, maxY float64) {
	builder.WriteString("0\nSECTION\n2\nHEADER\n")
	builder.WriteString("9\n$ACADVER\n1\nAC1015\n")
	builder.WriteString("9\n$DWGCODEPAGE\n3\nANSI_1252\n")
	builder.WriteString("9\n$INSUNITS\n70\n4\n")
	builder.WriteString("9\n$MEASUREMENT\n70\n1\n")
	builder.WriteString("9\n$LUNITS\n70\n2\n")
	builder.WriteString("9\n$EXTMIN\n10\n")
	builder.WriteString(fmt.Sprintf("%.3f\n20\n%.3f\n30\n0.0\n", minX, minY))
	builder.WriteString("9\n$EXTMAX\n10\n")
	builder.WriteString(fmt.Sprintf("%.3f\n20\n%.3f\n30\n0.0\n", maxX, maxY))
	builder.WriteString("0\nENDSEC\n")
}

func writeDXFTables(builder *strings.Builder, pieces []map[string]any) {
	builder.WriteString("0\nSECTION\n2\nTABLES\n")
	builder.WriteString("0\nTABLE\n2\nLTYPE\n70\n2\n")
	builder.WriteString("0\nLTYPE\n2\nCONTINUOUS\n70\n0\n3\nSolid line\n72\n65\n73\n0\n40\n0.0\n")
	builder.WriteString("0\nLTYPE\n2\nDASHED\n70\n0\n3\nDashed __ __\n72\n65\n73\n2\n40\n12.0\n49\n6.0\n49\n-6.0\n")
	builder.WriteString("0\nENDTAB\n")

	layerCount := 5 + len(pieces)*2
	builder.WriteString("0\nTABLE\n2\nLAYER\n70\n")
	builder.WriteString(fmt.Sprintf("%d\n", layerCount))
	writeLayer(builder, "ANNOTATION", 7, "CONTINUOUS")
	writeLayer(builder, "SEAM_ALLOWANCE", 3, "DASHED")
	writeLayer(builder, "GRAIN_LINE", 5, "DASHED")
	writeLayer(builder, "NOTCH", 1, "CONTINUOUS")
	writeLayer(builder, "DRILL", 2, "CONTINUOUS")
	for _, piece := range pieces {
		name := dxfLayerName(fmt.Sprint(piece["name"]))
		writeLayer(builder, name, 7, "CONTINUOUS")
		writeLayer(builder, name+"_PTS", 8, "CONTINUOUS")
	}
	builder.WriteString("0\nENDTAB\n")

	builder.WriteString("0\nTABLE\n2\nSTYLE\n70\n1\n")
	builder.WriteString("0\nSTYLE\n2\nSTANDARD\n70\n0\n40\n0.0\n41\n1.0\n50\n0.0\n71\n0\n42\n2.5\n3\ntxt\n4\n\n")
	builder.WriteString("0\nENDTAB\n")
	builder.WriteString("0\nENDSEC\n")
}

func writeLayer(builder *strings.Builder, name string, color int, ltype string) {
	builder.WriteString("0\nLAYER\n2\n")
	builder.WriteString(name)
	builder.WriteString("\n70\n0\n62\n")
	builder.WriteString(fmt.Sprintf("%d\n", color))
	builder.WriteString("6\n")
	builder.WriteString(ltype)
	builder.WriteString("\n")
}

func dxfLayerName(name string) string {
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_", "\\", "_", ":", "_")
	value := replacer.Replace(strings.ToUpper(strings.TrimSpace(asciiToken(name))))
	if value == "" {
		return "PATTERN"
	}
	return value
}

func asciiLabel(id, fallback string) string {
	if id = asciiToken(id); id != "" {
		return id
	}
	return asciiToken(fallback)
}

func asciiToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	mapped := map[string]string{
		"前片": "FRONT",
		"后片": "BACK",
		"长袖": "LONG_SLEEVE",
		"短袖": "SHORT_SLEEVE",
		"翻领": "TURN_DOWN_COLLAR",
		"立领": "STAND_COLLAR",
		"圆领": "ROUND_NECK",
		"V领": "V_NECK",
		"单排扣": "SINGLE_PLACKET",
		"贴袋": "PATCH_POCKET",
		"口袋-1": "POCKET_1",
		"口袋-2": "POCKET_2",
		"标准剪口": "STANDARD_NOTCH",
		"经向": "WARP",
		"纬向": "WEFT",
		"斜纹": "BIAS",
		"国标女": "CN_WOMENS",
		"国标男": "CN_MENS",
		"童装": "CHILDREN",
		"单扣": "SINGLE_BUTTON_CUFF",
		"双扣": "DOUBLE_BUTTON_CUFF",
		"罗纹": "RIB_CUFF",
		"button-count:6": "BUTTON_COUNT_6",
		"body": "BODY",
		"sleeve": "SLEEVE",
		"collar": "COLLAR",
		"placket": "PLACKET",
		"pocket": "POCKET",
	}
	if replacement, ok := mapped[trimmed]; ok {
		return replacement
	}

	var builder strings.Builder
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			continue
		}
		switch r {
		case ' ', '-', '/', '\\', ':', '.':
			builder.WriteRune('_')
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "ITEM"
	}
	return result
}

func writePolylineEntity(builder *strings.Builder, handleSeed *int, layer string, points []point, closed bool) {
	writeEntityPreamble(builder, handleSeed, "LWPOLYLINE", layer, "AcDbPolyline")
	builder.WriteString("90\n")
	builder.WriteString(fmt.Sprintf("%d\n", len(points)))
	flag := 0
	if closed {
		flag = 1
	}
	builder.WriteString(fmt.Sprintf("70\n%d\n", flag))
	builder.WriteString("43\n0.0\n38\n0.0\n39\n0.0\n")
	for _, item := range points {
		builder.WriteString("10\n")
		builder.WriteString(fmt.Sprintf("%.3f\n", item.X))
		builder.WriteString("20\n")
		builder.WriteString(fmt.Sprintf("%.3f\n", item.Y))
	}
}

func writePointEntity(builder *strings.Builder, handleSeed *int, layer string, value point) {
	writeEntityPreamble(builder, handleSeed, "POINT", layer, "AcDbPoint")
	builder.WriteString("10\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", value.X))
	builder.WriteString("20\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", value.Y))
	builder.WriteString("30\n0.0\n")
}

func writeCrossEntity(builder *strings.Builder, handleSeed *int, layer string, center point, size float64) {
	half := size / 2.0
	writeLineEntity(builder, handleSeed, layer, point{X: center.X - half, Y: center.Y}, point{X: center.X + half, Y: center.Y})
	writeLineEntity(builder, handleSeed, layer, point{X: center.X, Y: center.Y - half}, point{X: center.X, Y: center.Y + half})
}

func writeArrowEntity(builder *strings.Builder, handleSeed *int, layer string, tip, tail point) {
	dx := tail.X - tip.X
	dy := tail.Y - tip.Y
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	ux := dx / length
	uy := dy / length
	base := point{X: tip.X + ux*18, Y: tip.Y + uy*18}
	left := point{X: base.X - uy*6, Y: base.Y + ux*6}
	right := point{X: base.X + uy*6, Y: base.Y - ux*6}
	writeLineEntity(builder, handleSeed, layer, tip, left)
	writeLineEntity(builder, handleSeed, layer, tip, right)
}

func drawingBounds(pieces []map[string]any) (float64, float64, float64, float64) {
	minX := math.MaxFloat64
	minY := math.MaxFloat64
	maxX := -math.MaxFloat64
	maxY := -math.MaxFloat64
	for _, piece := range pieces {
		for _, item := range decodePoints(piece["points"]) {
			if item.X < minX {
				minX = item.X
			}
			if item.Y < minY {
				minY = item.Y
			}
			if item.X > maxX {
				maxX = item.X
			}
			if item.Y > maxY {
				maxY = item.Y
			}
		}
	}
	if minX == math.MaxFloat64 {
		return 0, 0, 1000, 1000
	}
	padding := 80.0
	return minX - padding, minY - padding, maxX + padding, maxY + padding
}

func expandPolygon(points []point, offset float64) []point {
	center := centroid(points)
	expanded := make([]point, 0, len(points))
	for _, item := range points {
		dx := item.X - center.X
		dy := item.Y - center.Y
		length := math.Hypot(dx, dy)
		if length == 0 {
			expanded = append(expanded, point{X: item.X + offset, Y: item.Y + offset})
			continue
		}
		scale := (length + offset) / length
		expanded = append(expanded, point{X: center.X + dx*scale, Y: center.Y + dy*scale})
	}
	return expanded
}

func grainLine(points []point) (point, point) {
	minX := points[0].X
	maxX := points[0].X
	minY := points[0].Y
	maxY := points[0].Y
	for _, item := range points[1:] {
		if item.X < minX {
			minX = item.X
		}
		if item.X > maxX {
			maxX = item.X
		}
		if item.Y < minY {
			minY = item.Y
		}
		if item.Y > maxY {
			maxY = item.Y
		}
	}
	midX := (minX + maxX) / 2.0
	return point{X: midX, Y: minY + 20}, point{X: midX, Y: maxY - 20}
}

func decodePoints(raw any) []point {
	items, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]point); ok {
			return typed
		}
		return nil
	}

	points := make([]point, 0, len(items))
	for _, item := range items {
		piecePoint, ok := item.(map[string]any)
		if !ok {
			continue
		}
		points = append(points, point{X: numericValue(piecePoint["x"], 0), Y: numericValue(piecePoint["y"], 0)})
	}
	return points
}

func writeLineEntity(builder *strings.Builder, handleSeed *int, layer string, from, to point) {
	writeEntityPreamble(builder, handleSeed, "LINE", layer, "AcDbLine")
	builder.WriteString("10\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", from.X))
	builder.WriteString("20\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", from.Y))
	builder.WriteString("30\n0.0\n11\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", to.X))
	builder.WriteString("21\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", to.Y))
	builder.WriteString("31\n0.0\n")
}

func writeTextEntity(builder *strings.Builder, handleSeed *int, layer, value string, x, y float64) {
	writeEntityPreamble(builder, handleSeed, "TEXT", layer, "AcDbText")
	builder.WriteString("10\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", x))
	builder.WriteString("20\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", y))
	builder.WriteString("30\n0.0\n40\n12.0\n1\n")
	builder.WriteString(value)
	builder.WriteString("\n7\nSTANDARD\n72\n0\n73\n0\n11\n")
	builder.WriteString(fmt.Sprintf("%.3f\n21\n%.3f\n31\n0.0\n", x, y))
}

func writeCircleEntity(builder *strings.Builder, handleSeed *int, layer string, center point, radius float64) {
	writeEntityPreamble(builder, handleSeed, "CIRCLE", layer, "AcDbCircle")
	builder.WriteString("10\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", center.X))
	builder.WriteString("20\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", center.Y))
	builder.WriteString("30\n0.0\n40\n")
	builder.WriteString(fmt.Sprintf("%.3f\n", radius))
}

func writeEntityPreamble(builder *strings.Builder, handleSeed *int, entityType, layer, subclass string) {
	builder.WriteString("0\n")
	builder.WriteString(entityType)
	builder.WriteString("\n5\n")
	builder.WriteString(fmt.Sprintf("%X\n", *handleSeed))
	*handleSeed++
	builder.WriteString("100\nAcDbEntity\n8\n")
	builder.WriteString(layer)
	builder.WriteString("\n6\nBYLAYER\n62\n256\n67\n0\n410\nModel\n")
	builder.WriteString("100\n")
	builder.WriteString(subclass)
	builder.WriteString("\n")
}

func polygonArea(points []point) float64 {
	if len(points) < 3 {
		return 0
	}
	area := 0.0
	for index := range points {
		next := (index + 1) % len(points)
		area += points[index].X*points[next].Y - points[next].X*points[index].Y
	}
	return math.Abs(area) * 0.5
}

func centroid(points []point) point {
	if len(points) == 0 {
		return point{}
	}
	xTotal := 0.0
	yTotal := 0.0
	for _, item := range points {
		xTotal += item.X
		yTotal += item.Y
	}
	return point{X: xTotal / float64(len(points)), Y: yTotal / float64(len(points))}
}

func mustJSON(data map[string]any) []byte {
	canonical := make(map[string]any, len(data))
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		canonical[key] = data[key]
	}
	payload, _ := json.Marshal(canonical)
	return payload
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}