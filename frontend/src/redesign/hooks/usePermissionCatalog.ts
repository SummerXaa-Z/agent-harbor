import { useEffect, useState } from "react";
import { fetchPermissionPackageAccessSubjects, fetchPermissionPackageTemplates } from "../../api";
import { accessSubjectOptions, normalizeAccessSubjectOptions, type AccessSubjectOption } from "../../accessSubjects";
import { permissionPackageTemplates, type PermissionPackageTemplate } from "../../permissionPackages";

export interface PermissionCatalog {
  subjects: AccessSubjectOption[];
  templates: PermissionPackageTemplate[];
}

// Templates and access subjects are read-only catalogs; the built-in copies
// stand in until the live ones arrive (or when the API is unreachable).
export function usePermissionCatalog(live: boolean): PermissionCatalog {
  const [templates, setTemplates] = useState<PermissionPackageTemplate[]>(permissionPackageTemplates);
  const [subjects, setSubjects] = useState<AccessSubjectOption[]>(accessSubjectOptions);

  useEffect(() => {
    if (!live) return;
    const controller = new AbortController();
    fetchPermissionPackageTemplates("", controller.signal)
      .then((rows) => {
        if (rows.length > 0) setTemplates(rows);
      })
      .catch(() => undefined);
    fetchPermissionPackageAccessSubjects("", controller.signal)
      .then((rows) => setSubjects(normalizeAccessSubjectOptions(rows)))
      .catch(() => undefined);
    return () => controller.abort();
  }, [live]);

  return { subjects, templates };
}
