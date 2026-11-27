// vite build 会清空 dist，把占位文件补回去：Go 的 //go:embed all:dist 要求目录里至少有一个文件，
// 没有构建前端时 go build 也能通过
import { writeFileSync } from "node:fs";

writeFileSync(new URL("../dist/.gitkeep", import.meta.url), "");
