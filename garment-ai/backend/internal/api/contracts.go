package api

type PhotoPayload struct {
	View        string `json:"view" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	Base64Data  string `json:"base64Data" binding:"required"`
}

type UploadPhotosRequest struct {
	SessionID string         `json:"sessionId" binding:"required"`
	Photos    []PhotoPayload `json:"photos" binding:"required,min=2,max=5"`
}

type GenerateParamsRequest struct {
	Structure map[string]any `json:"structure" binding:"required"`
	Ratios    map[string]any `json:"ratios" binding:"required"`
	TemplateID string        `json:"templateId"`
}

type GeneratePreviewRequest struct {
	Params map[string]any `json:"params" binding:"required"`
}

type GeneratePatternRequest struct {
	Params map[string]any `json:"params" binding:"required"`
}

func sampleGarmentParams() map[string]any {
	return map[string]any{
		"garment_type":      "shirt",
		"silhouette":        "H型",
		"length":            65.0,
		"chest":             96.0,
		"shoulder":          42.0,
		"neck":              38.0,
		"sleeve_length":     60.0,
		"cuff":              22.0,
		"waist":             84.0,
		"hip":               98.0,
		"collar_type":       "翻领",
		"collar_width":      8.0,
		"collar_depth":      5.0,
		"sleeve_type":       "长袖",
		"sleeve_cuff_type":  "单扣",
		"placket_type":      "单排扣",
		"placket_length":    20.0,
		"button_count":      6,
		"pocket_type":       "贴袋",
		"pocket_count":      2,
		"pocket_width":      12.0,
		"pocket_height":     14.0,
		"pocket_position_x": 15.0,
		"pocket_position_y": 25.0,
		"dart_position":     "前腰",
		"dart_count":        2,
		"dart_length":       10.0,
		"dart_width":        2.0,
		"seam_allowance":    1.0,
		"notch_type":        "标准剪口",
		"grain_line":        "经向",
		"grade_rule":        "国标女",
	}
}

func sampleStructure() map[string]any {
	return map[string]any{
		"garment_type": "shirt",
		"silhouette":   "H型",
		"collar_type":  "翻领",
		"sleeve_type":  "长袖",
		"placket_type": "单排扣",
		"pocket_type":  "贴袋",
		"pocket_count": 2,
		"dart_position": "前腰",
		"dart_count":   2,
		"has_belt":     false,
		"has_zipper":   false,
		"seam_type":    "平缝",
		"confidence":   0.92,
	}
}

func sampleRatios() map[string]any {
	return map[string]any{
		"shoulder_to_chest": 0.65,
		"length_to_height":  0.45,
		"sleeve_to_arm":     0.85,
		"collar_to_shoulder": 0.35,
	}
}

func sampleKeypoints() map[string]any {
	return map[string]any{
		"left_shoulder": []float64{120.0, 160.0},
		"right_shoulder": []float64{340.0, 162.0},
		"neck_point": []float64{230.0, 130.0},
		"chest_point": []float64{232.0, 250.0},
		"waist_point": []float64{232.0, 360.0},
		"hip_point": []float64{232.0, 470.0},
		"left_cuff": []float64{92.0, 540.0},
		"right_cuff": []float64{376.0, 538.0},
		"collar_left": []float64{188.0, 144.0},
		"collar_right": []float64{274.0, 144.0},
	}
}