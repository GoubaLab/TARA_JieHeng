package assurance

import (
	"fmt"
	"strings"
)

func reportLevel(method Object, field string, value any, offset int) string {
	if value == nil {
		return "待评估"
	}
	levels := libRows(method[field])
	index := libInt(value) - offset
	if index < 0 || index >= len(levels) {
		return "待评估"
	}
	return libStr(levels[index]["label"])
}

func reportMethodNote(method Object) string {
	parts := []string{libStr(method["name"]), "组织配置，非标准固定算法"}
	if method["impact_aggregation"] == "coefficient_sum" {
		coefficients, names := libObj(method["impact_coefficients"]), libObj(method["impact_dimensions"])
		values := []string{}
		for _, key := range impactKeys {
			values = append(values, libStr(names[key])+"×"+libStr(coefficients[key]))
		}
		parts = append(parts, "SFOP总分为各维等级乘系数之和："+strings.Join(values, " + ")+"；不归一化、不取整；零系数维度不参与评分")
	} else if method["impact_aggregation"] == "weighted_mean" {
		weights, names := libObj(method["impact_weights"]), libObj(method["impact_dimensions"])
		values := []string{}
		for _, key := range impactKeys {
			values = append(values, libStr(names[key])+"="+libStr(weights[key]))
		}
		parts = append(parts, "影响取加权平均并向上取整；权重为零的维度不参与评分；相对权重："+strings.Join(values, "、"))
	} else {
		parts = append(parts, "影响取四维最高等级")
	}
	if method["impact_aggregation"] == "coefficient_sum" {
		parts = append(parts, "风险分值=SFOP总分×可行性等级；直接使用原始总分，不加1；按阈值上限（含等号）映射为1至5级："+libJoin(method["product_thresholds"], "、"))
	} else if method["risk_calculation"] == "product" {
		parts = append(parts, "风险乘积=(影响等级+1)×可行性等级；按阈值上限（含等号）映射为1至5级："+libJoin(method["product_thresholds"], "、"))
	} else {
		parts = append(parts, "风险按下表矩阵查值")
	}
	note := strings.Join(parts, "；") + "。残余风险使用相同风险分级算法，输入来自独立残余评估。"
	for _, group := range [][2]string{{"impact_levels", "影响等级"}, {"feasibility_levels", "可行性等级"}, {"risk_levels", "风险等级"}} {
		for i, level := range libRows(method[group[0]]) {
			n := i + 1
			if group[0] == "impact_levels" {
				n = i
			}
			description := libStr(level["description"])
			if description == "" {
				description = "未设置判定锚点"
			}
			note += fmt.Sprintf("\n%s %d · %s：%s", group[1], n, libStr(level["label"]), description)
		}
	}
	return note
}

func reportReference(value any, catalog *Catalog) string {
	key := libStr(value)
	if key == "" || catalog == nil {
		return key
	}
	if entry, err := catalog.Get(key); err == nil && libBool(entry["withdrawn"]) {
		return key + "（来源已撤下）"
	}
	return key
}

func reportReferences(values any, catalog *Catalog) string {
	result := []string{}
	for _, value := range libList(values) {
		result = append(result, reportReference(value, catalog))
	}
	return strings.Join(result, "、")
}

func reportOS(value any) string {
	os := libObj(value)
	if len(os) == 0 {
		return "操作系统：未指定"
	}
	labels := map[string]string{"unknown": "未指定", "bare_metal": "裸机", "rtos": "实时操作系统", "linux": "Linux", "qnx": "QNX", "windows": "Windows", "android": "Android", "ios": "iOS", "macos": "macOS", "other": "其他"}
	parts := []string{"操作系统：" + labels[libStr(libFirst(os["family"], "unknown"))]}
	for _, pair := range [][2]string{{"name", "名称"}, {"version", "版本"}, {"architecture", "架构"}, {"notes", "说明"}} {
		if libBool(os[pair[0]]) {
			parts = append(parts, pair[1]+"："+libStr(os[pair[0]]))
		}
	}
	return strings.Join(parts, "；")
}

func reportCommunication(flow Object) string {
	c := libObj(flow["communication"])
	medium := map[string]string{"unspecified": "未指定", "wired": "有线", "wireless": "无线", "logical": "逻辑连接"}[libStr(libFirst(c["medium"], "unspecified"))]
	reach := map[string]string{"unspecified": "未指定", "near_field": "近场", "local": "局域", "wide_area": "广域"}[libStr(libFirst(c["range"], "unspecified"))]
	properties := []string{}
	for _, v := range libList(c["security_properties"]) {
		properties = append(properties, reportProperties[libStr(v)])
	}
	if len(properties) == 0 {
		properties = append(properties, "未声明")
	}
	return fmt.Sprintf("通信介质：%s；范围：%s\n技术：%s；应用协议：%s\n用途：%s\n安全属性：%s\n%s", medium, reach, libStr(c["technology"]), libStr(flow["protocol"]), libStr(c["purpose"]), strings.Join(properties, "、"), libStr(flow["description"]))
}
