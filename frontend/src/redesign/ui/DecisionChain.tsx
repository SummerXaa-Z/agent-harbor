import { Check, Clock, X } from "lucide-react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import type { DecisionChain as DecisionChainModel, DecisionNodeState } from "../model/decisionChain";

const nodeIcon: Record<DecisionNodeState, typeof Check> = {
  failed: X,
  passed: Check,
  pending: Clock,
};

// Six decision layers in evaluation order. State is carried by shape, icon and
// text, never color alone: passed is a green circle with a check, the
// terminating node a red rounded square with an X, and the layers after it
// grey dashed circles with a clock.
export function DecisionChain({ chain, label }: { chain: DecisionChainModel; label: string }) {
  const { t } = useRedesignI18n();
  return (
    <ol aria-label={label} className="dchain">
      {chain.nodes.map((node) => {
        const Icon = nodeIcon[node.state];
        return (
          <li
            aria-current={node.state === "failed" ? "step" : undefined}
            className={`dnode dnode-${node.state}`}
            key={node.layer}
          >
            <span className="dnode-ico">
              <Icon aria-hidden="true" size={15} />
            </span>
            <span className="dnode-label">{t(node.labelKey)}</span>
            <span className="dnode-state">{t(`rd.chain.${node.state}`)}</span>
          </li>
        );
      })}
    </ol>
  );
}
