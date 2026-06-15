import { ArrowRight, Check } from "lucide-react";

import type { Translator } from "../consolePresenters";
import {
  productionJourneyStages,
  type ProductionJourney
} from "../productionJourney";

export function ProductionJourneyCheckpoint({
  journey,
  t
}: {
  journey: ProductionJourney;
  t: Translator;
}) {
  const completed = new Set(journey.completedStageKeys);

  return (
    <section className="production-journey-checkpoint" aria-label={t("productionJourney.aria")}>
      <div className="production-journey-copy">
        <span className="section-kicker">{t("productionJourney.kicker")}</span>
        <strong>{t(journey.nextActionKey)}</strong>
      </div>
      <ol className="production-journey-steps">
        {productionJourneyStages.map((stage) => {
          const complete = completed.has(stage.key);
          const current = journey.currentStageKey === stage.key;
          return (
            <li className={current ? "is-current" : complete ? "is-complete" : ""} key={stage.key}>
              <span aria-hidden="true">{complete ? <Check size={12} /> : null}</span>
              {t(stage.labelKey)}
            </li>
          );
        })}
      </ol>
      <a className="secondary-button production-journey-next" href={journey.nextActionHash}>
        <span>{t("productionJourney.nextLabel")}</span>
        <ArrowRight aria-hidden="true" size={14} />
      </a>
    </section>
  );
}
