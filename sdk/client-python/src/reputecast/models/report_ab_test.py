from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_ab_test_kind import ReportABTestKind
from ..models.report_ab_test_metric import ReportABTestMetric
from ..models.report_ab_test_outcome import ReportABTestOutcome
from ..models.report_ab_test_state import ReportABTestState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.report_ab_variant import ReportABVariant





T = TypeVar("T", bound="ReportABTest")



@_attrs_define
class ReportABTest:
    """ 
        Attributes:
            bar_max (float):
            default (str):
            kind (ReportABTestKind):
            label (str): subject line, content, send time
            metric (ReportABTestMetric):
            min_sample (int):
            remaining (int):
            significant (bool): Confidence reached the threshold, with enough recipients per variant
            state (ReportABTestState):
            threshold (float):
            variants (list[ReportABVariant]):
            window_hours (float):
            confidence (float | Unset):
            decided_after (float | Unset):
            decided_at (datetime.datetime | Unset):
            leader (str | Unset):
            lift (float | Unset):
            outcome (ReportABTestOutcome | Unset):
            p_value (float | Unset):
            rollout_at (datetime.datetime | Unset):
            runner_up (str | Unset):
            winner (str | Unset):
     """

    bar_max: float
    default: str
    kind: ReportABTestKind
    label: str
    metric: ReportABTestMetric
    min_sample: int
    remaining: int
    significant: bool
    state: ReportABTestState
    threshold: float
    variants: list[ReportABVariant]
    window_hours: float
    confidence: float | Unset = UNSET
    decided_after: float | Unset = UNSET
    decided_at: datetime.datetime | Unset = UNSET
    leader: str | Unset = UNSET
    lift: float | Unset = UNSET
    outcome: ReportABTestOutcome | Unset = UNSET
    p_value: float | Unset = UNSET
    rollout_at: datetime.datetime | Unset = UNSET
    runner_up: str | Unset = UNSET
    winner: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_ab_variant import ReportABVariant # noqa: PLC0415
        bar_max = self.bar_max

        default = self.default

        kind = self.kind.value

        label = self.label

        metric = self.metric.value

        min_sample = self.min_sample

        remaining = self.remaining

        significant = self.significant

        state = self.state.value

        threshold = self.threshold

        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        window_hours = self.window_hours

        confidence = self.confidence

        decided_after = self.decided_after

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        leader = self.leader

        lift = self.lift

        outcome: str | Unset = UNSET
        if not isinstance(self.outcome, Unset):
            outcome = self.outcome.value


        p_value = self.p_value

        rollout_at: str | Unset = UNSET
        if not isinstance(self.rollout_at, Unset):
            rollout_at = self.rollout_at.isoformat()

        runner_up = self.runner_up

        winner = self.winner


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "barMax": bar_max,
            "default": default,
            "kind": kind,
            "label": label,
            "metric": metric,
            "minSample": min_sample,
            "remaining": remaining,
            "significant": significant,
            "state": state,
            "threshold": threshold,
            "variants": variants,
            "windowHours": window_hours,
        })
        if confidence is not UNSET:
            field_dict["confidence"] = confidence
        if decided_after is not UNSET:
            field_dict["decidedAfter"] = decided_after
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if leader is not UNSET:
            field_dict["leader"] = leader
        if lift is not UNSET:
            field_dict["lift"] = lift
        if outcome is not UNSET:
            field_dict["outcome"] = outcome
        if p_value is not UNSET:
            field_dict["pValue"] = p_value
        if rollout_at is not UNSET:
            field_dict["rolloutAt"] = rollout_at
        if runner_up is not UNSET:
            field_dict["runnerUp"] = runner_up
        if winner is not UNSET:
            field_dict["winner"] = winner

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_ab_variant import ReportABVariant # noqa: PLC0415
        d = dict(src_dict)
        bar_max = d.pop("barMax")

        default = d.pop("default")

        kind = ReportABTestKind(d.pop("kind"))




        label = d.pop("label")

        metric = ReportABTestMetric(d.pop("metric"))




        min_sample = d.pop("minSample")

        remaining = d.pop("remaining")

        significant = d.pop("significant")

        state = ReportABTestState(d.pop("state"))




        threshold = d.pop("threshold")

        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = ReportABVariant.from_dict(variants_item_data)



            variants.append(variants_item)


        window_hours = d.pop("windowHours")

        confidence = d.pop("confidence", UNSET)

        decided_after = d.pop("decidedAfter", UNSET)

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        leader = d.pop("leader", UNSET)

        lift = d.pop("lift", UNSET)

        _outcome = d.pop("outcome", UNSET)
        outcome: ReportABTestOutcome | Unset
        if isinstance(_outcome,  Unset):
            outcome = UNSET
        else:
            outcome = ReportABTestOutcome(_outcome)




        p_value = d.pop("pValue", UNSET)

        _rollout_at = d.pop("rolloutAt", UNSET)
        rollout_at: datetime.datetime | Unset
        if isinstance(_rollout_at,  Unset):
            rollout_at = UNSET
        else:
            rollout_at = datetime.datetime.fromisoformat(_rollout_at)




        runner_up = d.pop("runnerUp", UNSET)

        winner = d.pop("winner", UNSET)

        report_ab_test = cls(
            bar_max=bar_max,
            default=default,
            kind=kind,
            label=label,
            metric=metric,
            min_sample=min_sample,
            remaining=remaining,
            significant=significant,
            state=state,
            threshold=threshold,
            variants=variants,
            window_hours=window_hours,
            confidence=confidence,
            decided_after=decided_after,
            decided_at=decided_at,
            leader=leader,
            lift=lift,
            outcome=outcome,
            p_value=p_value,
            rollout_at=rollout_at,
            runner_up=runner_up,
            winner=winner,
        )

        return report_ab_test

