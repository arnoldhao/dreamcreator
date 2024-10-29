package types

// CapCutContentMaterialsTextsContent CapCutContent.Materials.Texts.Content
type CapCutContentMaterialsTextsContent struct {
	Text string `json:"text"`
}

type TargetTimerange struct {
	Duration int `json:"duration"`
	Start    int `json:"start"`
}

// CapCutContent CapCut Content Allin1
type CapCutContent struct {
	CanvasConfig struct {
		Height int    `json:"height"`
		Ratio  string `json:"ratio"`
		Width  int    `json:"width"`
	} `json:"canvas_config"`
	ColorSpace int `json:"color_space"`
	Config     struct {
		AdjustMaxIndex         int    `json:"adjust_max_index"`
		AttachmentInfo         []any  `json:"attachment_info"`
		CombinationMaxIndex    int    `json:"combination_max_index"`
		ExportRange            any    `json:"export_range"`
		ExtractAudioLastIndex  int    `json:"extract_audio_last_index"`
		LyricsRecognitionID    string `json:"lyrics_recognition_id"`
		LyricsSync             bool   `json:"lyrics_sync"`
		LyricsTaskinfo         []any  `json:"lyrics_taskinfo"`
		MaintrackAdsorb        bool   `json:"maintrack_adsorb"`
		MaterialSaveMode       int    `json:"material_save_mode"`
		MultiLanguageCurrent   string `json:"multi_language_current"`
		MultiLanguageList      []any  `json:"multi_language_list"`
		MultiLanguageMain      string `json:"multi_language_main"`
		MultiLanguageMode      string `json:"multi_language_mode"`
		OriginalSoundLastIndex int    `json:"original_sound_last_index"`
		RecordAudioLastIndex   int    `json:"record_audio_last_index"`
		StickerMaxIndex        int    `json:"sticker_max_index"`
		SubtitleKeywordsConfig any    `json:"subtitle_keywords_config"`
		SubtitleRecognitionID  string `json:"subtitle_recognition_id"`
		SubtitleSync           bool   `json:"subtitle_sync"`
		SubtitleTaskinfo       []any  `json:"subtitle_taskinfo"`
		SystemFontList         []any  `json:"system_font_list"`
		VideoMute              bool   `json:"video_mute"`
		ZoomInfoParams         struct {
			OffsetX   float64 `json:"offset_x"`
			OffsetY   float64 `json:"offset_y"`
			ZoomRatio float64 `json:"zoom_ratio"`
		} `json:"zoom_info_params"`
	} `json:"config"`
	Cover                 any     `json:"cover"`
	CreateTime            int     `json:"create_time"`
	Duration              int     `json:"duration"`
	ExtraInfo             any     `json:"extra_info"`
	Fps                   float64 `json:"fps"`
	FreeRenderIndexModeOn bool    `json:"free_render_index_mode_on"`
	GroupContainer        any     `json:"group_container"`
	ID                    string  `json:"id"`
	KeyframeGraphList     []any   `json:"keyframe_graph_list"`
	Keyframes             struct {
		Adjusts    []any `json:"adjusts"`
		Audios     []any `json:"audios"`
		Effects    []any `json:"effects"`
		Filters    []any `json:"filters"`
		Handwrites []any `json:"handwrites"`
		Stickers   []any `json:"stickers"`
		Texts      []any `json:"texts"`
		Videos     []any `json:"videos"`
	} `json:"keyframes"`
	LastModifiedPlatform struct {
		AppID      int    `json:"app_id"`
		AppSource  string `json:"app_source"`
		AppVersion string `json:"app_version"`
		DeviceID   string `json:"device_id"`
		HardDiskID string `json:"hard_disk_id"`
		MacAddress string `json:"mac_address"`
		Os         string `json:"os"`
		OsVersion  string `json:"os_version"`
	} `json:"last_modified_platform"`
	Materials struct {
		AiTranslates  []any `json:"ai_translates"`
		AudioBalances []any `json:"audio_balances"`
		AudioEffects  []any `json:"audio_effects"`
		AudioFades    []struct {
			FadeInDuration  int    `json:"fade_in_duration"`
			FadeOutDuration int    `json:"fade_out_duration"`
			FadeType        int    `json:"fade_type"`
			ID              string `json:"id"`
			Type            string `json:"type"`
		} `json:"audio_fades"`
		AudioTrackIndexes []any `json:"audio_track_indexes"`
		Audios            []any `json:"audios"`
		Beats             []any `json:"beats"`
		Canvases          []struct {
			AlbumImage     string  `json:"album_image"`
			Blur           float64 `json:"blur"`
			Color          string  `json:"color"`
			ID             string  `json:"id"`
			Image          string  `json:"image"`
			ImageID        string  `json:"image_id"`
			ImageName      string  `json:"image_name"`
			SourcePlatform int     `json:"source_platform"`
			TeamID         string  `json:"team_id"`
			Type           string  `json:"type"`
		} `json:"canvases"`
		Chromas            []any `json:"chromas"`
		ColorCurves        []any `json:"color_curves"`
		DigitalHumans      []any `json:"digital_humans"`
		Drafts             []any `json:"drafts"`
		Effects            []any `json:"effects"`
		Flowers            []any `json:"flowers"`
		GreenScreens       []any `json:"green_screens"`
		Handwrites         []any `json:"handwrites"`
		Hsl                []any `json:"hsl"`
		Images             []any `json:"images"`
		LogColorWheels     []any `json:"log_color_wheels"`
		Loudnesses         []any `json:"loudnesses"`
		ManualDeformations []any `json:"manual_deformations"`
		Masks              []any `json:"masks"`
		MaterialAnimations []struct {
			Animations           []any  `json:"animations"`
			ID                   string `json:"id"`
			MultiLanguageCurrent string `json:"multi_language_current"`
			Type                 string `json:"type"`
		} `json:"material_animations"`
		MaterialColors       []any `json:"material_colors"`
		MultiLanguageRefs    []any `json:"multi_language_refs"`
		Placeholders         []any `json:"placeholders"`
		PluginEffects        []any `json:"plugin_effects"`
		PrimaryColorWheels   []any `json:"primary_color_wheels"`
		RealtimeDenoises     []any `json:"realtime_denoises"`
		Shapes               []any `json:"shapes"`
		SmartCrops           []any `json:"smart_crops"`
		SmartRelights        []any `json:"smart_relights"`
		SoundChannelMappings []struct {
			AudioChannelMapping int    `json:"audio_channel_mapping"`
			ID                  string `json:"id"`
			IsConfigOpen        bool   `json:"is_config_open"`
			Type                string `json:"type"`
		} `json:"sound_channel_mappings"`
		Speeds []struct {
			CurveSpeed any     `json:"curve_speed"`
			ID         string  `json:"id"`
			Mode       int     `json:"mode"`
			Speed      float64 `json:"speed"`
			Type       string  `json:"type"`
		} `json:"speeds"`
		Stickers []struct {
			AigcType        string  `json:"aigc_type"`
			BackgroundAlpha float64 `json:"background_alpha"`
			BackgroundColor string  `json:"background_color"`
			BorderColor     string  `json:"border_color"`
			BorderLineStyle int     `json:"border_line_style"`
			BorderWidth     float64 `json:"border_width"`
			CategoryID      string  `json:"category_id"`
			CategoryName    string  `json:"category_name"`
			CheckFlag       int     `json:"check_flag"`
			ComboInfo       struct {
				TextTemplates []any `json:"text_templates"`
			} `json:"combo_info"`
			CycleSetting         bool    `json:"cycle_setting"`
			FormulaID            string  `json:"formula_id"`
			GlobalAlpha          float64 `json:"global_alpha"`
			HasShadow            bool    `json:"has_shadow"`
			IconURL              string  `json:"icon_url"`
			ID                   string  `json:"id"`
			MultiLanguageCurrent string  `json:"multi_language_current"`
			Name                 string  `json:"name"`
			OriginalSize         []any   `json:"original_size"`
			Path                 string  `json:"path"`
			Platform             string  `json:"platform"`
			PreviewCoverURL      string  `json:"preview_cover_url"`
			Radius               struct {
				BottomLeft  float64 `json:"bottom_left"`
				BottomRight float64 `json:"bottom_right"`
				TopLeft     float64 `json:"top_left"`
				TopRight    float64 `json:"top_right"`
			} `json:"radius"`
			RequestID      string  `json:"request_id"`
			ResourceID     string  `json:"resource_id"`
			SequenceType   bool    `json:"sequence_type"`
			ShadowAlpha    float64 `json:"shadow_alpha"`
			ShadowAngle    float64 `json:"shadow_angle"`
			ShadowColor    string  `json:"shadow_color"`
			ShadowDistance float64 `json:"shadow_distance"`
			ShadowPoint    struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"shadow_point"`
			ShadowSmoothing float64 `json:"shadow_smoothing"`
			ShapeParam      struct {
				CustomPoints []any `json:"custom_points"`
				Roundness    []any `json:"roundness"`
				ShapeSize    []any `json:"shape_size"`
				ShapeType    int   `json:"shape_type"`
			} `json:"shape_param"`
			SourcePlatform int    `json:"source_platform"`
			StickerID      string `json:"sticker_id"`
			SubType        int    `json:"sub_type"`
			TeamID         string `json:"team_id"`
			Type           string `json:"type"`
			Unicode        string `json:"unicode"`
		} `json:"stickers"`
		TailLeaders   []any `json:"tail_leaders"`
		TextTemplates []any `json:"text_templates"`
		Texts         []struct {
			AddType                    int     `json:"add_type"`
			Alignment                  int     `json:"alignment"`
			BackgroundAlpha            float64 `json:"background_alpha"`
			BackgroundColor            string  `json:"background_color"`
			BackgroundHeight           float64 `json:"background_height"`
			BackgroundHorizontalOffset float64 `json:"background_horizontal_offset"`
			BackgroundRoundRadius      float64 `json:"background_round_radius"`
			BackgroundStyle            int     `json:"background_style"`
			BackgroundVerticalOffset   float64 `json:"background_vertical_offset"`
			BackgroundWidth            float64 `json:"background_width"`
			BaseContent                string  `json:"base_content"`
			BoldWidth                  float64 `json:"bold_width"`
			BorderAlpha                float64 `json:"border_alpha"`
			BorderColor                string  `json:"border_color"`
			BorderWidth                float64 `json:"border_width"`
			CaptionTemplateInfo        struct {
				CategoryID     string `json:"category_id"`
				CategoryName   string `json:"category_name"`
				EffectID       string `json:"effect_id"`
				IsNew          bool   `json:"is_new"`
				Path           string `json:"path"`
				RequestID      string `json:"request_id"`
				ResourceID     string `json:"resource_id"`
				ResourceName   string `json:"resource_name"`
				SourcePlatform int    `json:"source_platform"`
			} `json:"caption_template_info"`
			CheckFlag int `json:"check_flag"`
			ComboInfo struct {
				TextTemplates []any `json:"text_templates"`
			} `json:"combo_info"`
			Content                string  `json:"content"`
			FixedHeight            float64 `json:"fixed_height"`
			FixedWidth             float64 `json:"fixed_width"`
			FontCategoryID         string  `json:"font_category_id"`
			FontCategoryName       string  `json:"font_category_name"`
			FontID                 string  `json:"font_id"`
			FontName               string  `json:"font_name"`
			FontPath               string  `json:"font_path"`
			FontResourceID         string  `json:"font_resource_id"`
			FontSize               float64 `json:"font_size"`
			FontSourcePlatform     int     `json:"font_source_platform"`
			FontTeamID             string  `json:"font_team_id"`
			FontTitle              string  `json:"font_title"`
			FontURL                string  `json:"font_url"`
			Fonts                  []any   `json:"fonts"`
			ForceApplyLineMaxWidth bool    `json:"force_apply_line_max_width"`
			GlobalAlpha            float64 `json:"global_alpha"`
			GroupID                string  `json:"group_id"`
			HasShadow              bool    `json:"has_shadow"`
			ID                     string  `json:"id"`
			InitialScale           float64 `json:"initial_scale"`
			InnerPadding           float64 `json:"inner_padding"`
			IsRichText             bool    `json:"is_rich_text"`
			ItalicDegree           int     `json:"italic_degree"`
			KtvColor               string  `json:"ktv_color"`
			Language               string  `json:"language"`
			LayerWeight            int     `json:"layer_weight"`
			LetterSpacing          float64 `json:"letter_spacing"`
			LineFeed               int     `json:"line_feed"`
			LineMaxWidth           float64 `json:"line_max_width"`
			LineSpacing            float64 `json:"line_spacing"`
			MultiLanguageCurrent   string  `json:"multi_language_current"`
			Name                   string  `json:"name"`
			OriginalSize           []any   `json:"original_size"`
			PresetCategory         string  `json:"preset_category"`
			PresetCategoryID       string  `json:"preset_category_id"`
			PresetHasSetAlignment  bool    `json:"preset_has_set_alignment"`
			PresetID               string  `json:"preset_id"`
			PresetIndex            int     `json:"preset_index"`
			PresetName             string  `json:"preset_name"`
			RecognizeTaskID        string  `json:"recognize_task_id"`
			RecognizeType          int     `json:"recognize_type"`
			RelevanceSegment       []any   `json:"relevance_segment"`
			ShadowAlpha            float64 `json:"shadow_alpha"`
			ShadowAngle            float64 `json:"shadow_angle"`
			ShadowColor            string  `json:"shadow_color"`
			ShadowDistance         float64 `json:"shadow_distance"`
			ShadowPoint            struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"shadow_point"`
			ShadowSmoothing                  float64 `json:"shadow_smoothing"`
			ShapeClipX                       bool    `json:"shape_clip_x"`
			ShapeClipY                       bool    `json:"shape_clip_y"`
			SourceFrom                       string  `json:"source_from"`
			StyleName                        string  `json:"style_name"`
			SubType                          int     `json:"sub_type"`
			SubtitleKeywords                 any     `json:"subtitle_keywords"`
			SubtitleTemplateOriginalFontsize float64 `json:"subtitle_template_original_fontsize"`
			TextAlpha                        float64 `json:"text_alpha"`
			TextColor                        string  `json:"text_color"`
			TextCurve                        any     `json:"text_curve"`
			TextPresetResourceID             string  `json:"text_preset_resource_id"`
			TextSize                         int     `json:"text_size"`
			TextToAudioIds                   []any   `json:"text_to_audio_ids"`
			TtsAutoUpdate                    bool    `json:"tts_auto_update"`
			Type                             string  `json:"type"`
			Typesetting                      int     `json:"typesetting"`
			Underline                        bool    `json:"underline"`
			UnderlineOffset                  float64 `json:"underline_offset"`
			UnderlineWidth                   float64 `json:"underline_width"`
			UseEffectDefaultColor            bool    `json:"use_effect_default_color"`
			Words                            struct {
				EndTime   []any `json:"end_time"`
				StartTime []any `json:"start_time"`
				Text      []any `json:"text"`
			} `json:"words"`
		} `json:"texts"`
		TimeMarks []struct {
			ID        string `json:"id"`
			MarkItems []any  `json:"mark_items"`
		} `json:"time_marks"`
		Transitions []struct {
			CategoryID   string `json:"category_id"`
			CategoryName string `json:"category_name"`
			Duration     int    `json:"duration"`
			EffectID     string `json:"effect_id"`
			ID           string `json:"id"`
			IsOverlap    bool   `json:"is_overlap"`
			Name         string `json:"name"`
			Path         string `json:"path"`
			Platform     string `json:"platform"`
			RequestID    string `json:"request_id"`
			ResourceID   string `json:"resource_id"`
			Type         string `json:"type"`
		} `json:"transitions"`
		VideoEffects   []any `json:"video_effects"`
		VideoTrackings []any `json:"video_trackings"`
		Videos         []struct {
			AigcType     string `json:"aigc_type"`
			AudioFade    any    `json:"audio_fade"`
			CartoonPath  string `json:"cartoon_path"`
			CategoryID   string `json:"category_id"`
			CategoryName string `json:"category_name"`
			CheckFlag    int    `json:"check_flag"`
			Crop         struct {
				LowerLeftX  float64 `json:"lower_left_x"`
				LowerLeftY  float64 `json:"lower_left_y"`
				LowerRightX float64 `json:"lower_right_x"`
				LowerRightY float64 `json:"lower_right_y"`
				UpperLeftX  float64 `json:"upper_left_x"`
				UpperLeftY  float64 `json:"upper_left_y"`
				UpperRightX float64 `json:"upper_right_x"`
				UpperRightY float64 `json:"upper_right_y"`
			} `json:"crop"`
			CropRatio            string  `json:"crop_ratio"`
			CropScale            float64 `json:"crop_scale"`
			Duration             int     `json:"duration"`
			ExtraTypeOption      int     `json:"extra_type_option"`
			FormulaID            string  `json:"formula_id"`
			Freeze               any     `json:"freeze"`
			HasAudio             bool    `json:"has_audio"`
			Height               int     `json:"height"`
			ID                   string  `json:"id"`
			IntensifiesAudioPath string  `json:"intensifies_audio_path"`
			IntensifiesPath      string  `json:"intensifies_path"`
			IsAiGenerateContent  bool    `json:"is_ai_generate_content"`
			IsCopyright          bool    `json:"is_copyright"`
			IsTextEditOverdub    bool    `json:"is_text_edit_overdub"`
			IsUnifiedBeautyMode  bool    `json:"is_unified_beauty_mode"`
			LocalID              string  `json:"local_id"`
			LocalMaterialID      string  `json:"local_material_id"`
			MaterialID           string  `json:"material_id"`
			MaterialName         string  `json:"material_name"`
			MaterialURL          string  `json:"material_url"`
			Matting              struct {
				Flag              int    `json:"flag"`
				HasUseQuickBrush  bool   `json:"has_use_quick_brush"`
				HasUseQuickEraser bool   `json:"has_use_quick_eraser"`
				InteractiveTime   []any  `json:"interactiveTime"`
				Path              string `json:"path"`
				Strokes           []any  `json:"strokes"`
			} `json:"matting"`
			MediaPath              string `json:"media_path"`
			ObjectLocked           any    `json:"object_locked"`
			OriginMaterialID       string `json:"origin_material_id"`
			Path                   string `json:"path"`
			PictureFrom            string `json:"picture_from"`
			PictureSetCategoryID   string `json:"picture_set_category_id"`
			PictureSetCategoryName string `json:"picture_set_category_name"`
			RequestID              string `json:"request_id"`
			ReverseIntensifiesPath string `json:"reverse_intensifies_path"`
			ReversePath            string `json:"reverse_path"`
			SmartMotion            any    `json:"smart_motion"`
			Source                 int    `json:"source"`
			SourcePlatform         int    `json:"source_platform"`
			Stable                 struct {
				MatrixPath  string `json:"matrix_path"`
				StableLevel int    `json:"stable_level"`
				TimeRange   struct {
					Duration int `json:"duration"`
					Start    int `json:"start"`
				} `json:"time_range"`
			} `json:"stable"`
			TeamID         string `json:"team_id"`
			Type           string `json:"type"`
			VideoAlgorithm struct {
				Algorithms            []any  `json:"algorithms"`
				ComplementFrameConfig any    `json:"complement_frame_config"`
				Deflicker             any    `json:"deflicker"`
				GameplayConfigs       []any  `json:"gameplay_configs"`
				MotionBlurConfig      any    `json:"motion_blur_config"`
				NoiseReduction        any    `json:"noise_reduction"`
				Path                  string `json:"path"`
				QualityEnhance        any    `json:"quality_enhance"`
				TimeRange             any    `json:"time_range"`
			} `json:"video_algorithm"`
			Width int `json:"width"`
		} `json:"videos"`
		VocalBeautifys   []any `json:"vocal_beautifys"`
		VocalSeparations []struct {
			Choice         int    `json:"choice"`
			ID             string `json:"id"`
			ProductionPath string `json:"production_path"`
			TimeRange      any    `json:"time_range"`
			Type           string `json:"type"`
		} `json:"vocal_separations"`
	} `json:"materials"`
	MutableConfig any    `json:"mutable_config"`
	Name          string `json:"name"`
	NewVersion    string `json:"new_version"`
	Platform      struct {
		AppID      int    `json:"app_id"`
		AppSource  string `json:"app_source"`
		AppVersion string `json:"app_version"`
		DeviceID   string `json:"device_id"`
		HardDiskID string `json:"hard_disk_id"`
		MacAddress string `json:"mac_address"`
		Os         string `json:"os"`
		OsVersion  string `json:"os_version"`
	} `json:"platform"`
	Relationships          []any  `json:"relationships"`
	RenderIndexTrackModeOn bool   `json:"render_index_track_mode_on"`
	RetouchCover           any    `json:"retouch_cover"`
	Source                 string `json:"source"`
	StaticCoverImagePath   string `json:"static_cover_image_path"`
	TimeMarks              any    `json:"time_marks"`
	Tracks                 []struct {
		Attribute     int    `json:"attribute"`
		Flag          int    `json:"flag"`
		ID            string `json:"id"`
		IsDefaultName bool   `json:"is_default_name"`
		Name          string `json:"name"`
		Segments      []struct {
			CaptionInfo any  `json:"caption_info"`
			Cartoon     bool `json:"cartoon"`
			Clip        struct {
				Alpha float64 `json:"alpha"`
				Flip  struct {
					Horizontal bool `json:"horizontal"`
					Vertical   bool `json:"vertical"`
				} `json:"flip"`
				Rotation float64 `json:"rotation"`
				Scale    struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"scale"`
				Transform struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"transform"`
			} `json:"clip"`
			CommonKeyframes        []any    `json:"common_keyframes"`
			EnableAdjust           bool     `json:"enable_adjust"`
			EnableColorCurves      bool     `json:"enable_color_curves"`
			EnableColorMatchAdjust bool     `json:"enable_color_match_adjust"`
			EnableColorWheels      bool     `json:"enable_color_wheels"`
			EnableLut              bool     `json:"enable_lut"`
			EnableSmartColorAdjust bool     `json:"enable_smart_color_adjust"`
			ExtraMaterialRefs      []string `json:"extra_material_refs"`
			GroupID                string   `json:"group_id"`
			HdrSettings            struct {
				Intensity float64 `json:"intensity"`
				Mode      int     `json:"mode"`
				Nits      int     `json:"nits"`
			} `json:"hdr_settings"`
			ID                string  `json:"id"`
			IntensifiesAudio  bool    `json:"intensifies_audio"`
			IsPlaceholder     bool    `json:"is_placeholder"`
			IsToneModify      bool    `json:"is_tone_modify"`
			KeyframeRefs      []any   `json:"keyframe_refs"`
			LastNonzeroVolume float64 `json:"last_nonzero_volume"`
			MaterialID        string  `json:"material_id"`
			RenderIndex       int     `json:"render_index"`
			ResponsiveLayout  struct {
				Enable              bool   `json:"enable"`
				HorizontalPosLayout int    `json:"horizontal_pos_layout"`
				SizeLayout          int    `json:"size_layout"`
				TargetFollow        string `json:"target_follow"`
				VerticalPosLayout   int    `json:"vertical_pos_layout"`
			} `json:"responsive_layout"`
			Reverse         bool `json:"reverse"`
			SourceTimerange struct {
				Duration int `json:"duration"`
				Start    int `json:"start"`
			} `json:"source_timerange"`
			Speed           float64 `json:"speed"`
			TargetTimerange struct {
				Duration int `json:"duration"`
				Start    int `json:"start"`
			} `json:"target_timerange"`
			TemplateID       string `json:"template_id"`
			TemplateScene    string `json:"template_scene"`
			TrackAttribute   int64  `json:"track_attribute"`
			TrackRenderIndex int    `json:"track_render_index"`
			UniformScale     struct {
				On    bool    `json:"on"`
				Value float64 `json:"value"`
			} `json:"uniform_scale"`
			Visible bool    `json:"visible"`
			Volume  float64 `json:"volume"`
		} `json:"segments"`
		Type string `json:"type"`
	} `json:"tracks"`
	UpdateTime int `json:"update_time"`
	Version    int `json:"version"`
}
