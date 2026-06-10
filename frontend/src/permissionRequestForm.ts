import type { AccessSubjectOption } from "./accessSubjects";
import type { PermissionPackageDraftInput } from "./permissionPackages";

export function applyPermissionRequestAccessSubject(
  form: PermissionPackageDraftInput,
  accessSubjectCatalog: AccessSubjectOption[],
  accessSubjectId: string
): PermissionPackageDraftInput {
  const option = accessSubjectCatalog.find((item) => item.id === accessSubjectId);
  if (!option || option.id === "custom" || option.subjectSelector === form.subjectSelector) {
    return form;
  }
  return { ...form, subjectSelector: option.subjectSelector };
}
