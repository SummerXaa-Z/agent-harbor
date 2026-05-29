# Frontend Design Reference

本文记录 AI Nexus Go Rebirth 前端的 clean-room 设计原则。它只描述产品体验与信息架构方向，不引用、不复制旧 AI Nexus `web/` 的源码、样式、组件结构或生成资产。

## 参考来源

可参考公开的企业后台与组件体系，但只能吸收通用设计逻辑：

- Ant Design Pro: https://pro.ant.design/
- Semi Design: https://semi.design/
- shadcn/ui dashboard blocks: https://ui.shadcn.com/blocks?category=dashboard
- Arco Design Pro: https://github.com/arco-design/arco-design-pro

这些来源共同体现了中后台产品的常见模式：稳定的应用壳、左侧导航、顶部状态区、表格/表单/详情页组合、主题 token、可访问性、mock 数据与真实 API 的切换能力。实现时应重新设计组件、布局、命名与交互细节。

## 产品气质

前端应是 To B 企业控制台，而不是营销页、官网首页或宣传型 demo。第一屏直接进入可操作的工作台，优先服务网关管理员、平台安全负责人、集成工程师和审计人员的日常工作。

界面表达应克制、清晰、可扫描：用密集但有秩序的信息排布展示 agent、credential、grant、policy、route、trace、evidence 等对象。避免大幅 hero、夸张渐变、装饰插画、过大的卡片堆叠和宣传口号。

## 信息架构

- 使用左侧主导航承载核心模块：Overview、Agents、Providers、Access Grants、Policies、Routes、Audit Traces、Settings。
- 顶部环境栏展示当前环境、租户/工作区、API base、后端连接状态、mock fallback 状态和用户入口。
- 列表页以表格为主，提供状态筛选、搜索、排序、批量操作、行内快捷动作和详情抽屉。
- 详情页应围绕对象生命周期组织：基础信息、配置、权限、调用记录、风险提示、变更历史。
- 控制台应支持深链接和稳定 URL，便于排障、协作和审计引用。

## 策略 / 审计 / 证据闭环

AI Nexus 的核心不是“好看的 Agent 列表”，而是治理闭环。所有关键页面都应能回答三类问题：

- 策略：谁可以调用谁，以什么协议、方法、路径和条件调用。
- 审计：调用何时发生，由谁发起，命中了什么 grant/policy，结果是 allowed 还是 denied。
- 证据：每一次决策要能追溯到 request 摘要、route、policy 版本、trace id、错误码和下游响应摘要。

视觉上应把 allowed/denied、active/draft/revoked、healthy/degraded/unreachable 等状态做成稳定一致的状态语言。危险操作必须有明确确认、可撤销预期或审计记录。

## 可访问性与可维护性

- 键盘可达：导航、表格行操作、弹窗、抽屉和表单控件都应支持合理的 tab 顺序与焦点管理。
- 语义清晰：按钮、表单项、状态提示、错误信息和空状态要能被辅助技术理解。
- 对比度足够：状态色不能只靠颜色表达，必要时配合文字、图标或 badge。
- 响应式策略以工作台为中心：桌面优先，同时保证窄屏下可完成查看、筛选和关键操作。
- 使用 token 化的颜色、间距、字号和阴影，减少一次性样式；新增页面应复用本仓库前端自己的组件模式。

## Clean-room 约束

- 不复制旧 AI Nexus `web/` 目录中的源码、CSS、组件结构、路由配置、mock 数据或图片资源。
- 不按旧实现逐文件翻译，也不保留旧文件命名作为迁移痕迹。
- 可以依据公开协议、README、clean-room spec、API 行为和公开设计系统建立新的信息架构。
- 设计参考只用于产品级模式，不用于粘贴模板代码；如引入第三方 UI 库或 blocks，必须遵守其 license，并在本项目中重新适配。
