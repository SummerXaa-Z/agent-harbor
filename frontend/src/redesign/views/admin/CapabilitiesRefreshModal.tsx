import { Button } from "../../ui/Button";
import { Modal } from "../../ui/Modal";
import type { useRedesignI18n } from "../../hooks/useRedesignI18n";

export function CapabilitiesRefreshModal({
  open,
  acting,
  t,
  onCancel,
  onConfirm
}: {
  open: boolean;
  acting: boolean;
  t: ReturnType<typeof useRedesignI18n>["t"];
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <Modal
      footer={
        <>
          <Button onClick={onCancel} variant="ghost">{t("rd.apr.cancel")}</Button>
          <Button disabled={acting} onClick={onConfirm} variant="primary">
            {t("rd.cap.refreshAction")}
          </Button>
        </>
      }
      onClose={onCancel}
      open={open}
      size="narrow"
      title={t("rd.cap.refreshTitle")}
    >
      <p className="muted">{t("rd.cap.refreshConfirm")}</p>
    </Modal>
  );
}
