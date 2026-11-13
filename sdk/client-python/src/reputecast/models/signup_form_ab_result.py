from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_ab_result_outcome import SignupFormABResultOutcome
from ..models.signup_form_ab_result_status import SignupFormABResultStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.signup_form_ab_variant import SignupFormABVariant





T = TypeVar("T", bound="SignupFormABResult")



@_attrs_define
class SignupFormABResult:
    """ 
        Attributes:
            auto_end (bool):
            confidence (float):
            double_opt_in (bool):
            fewest_views (int):
            lift (float):
            min_views (int):
            status (SignupFormABResultStatus):
            target_confidence (float): What a winner needs, 0.95 for 95%
            variants (list[SignupFormABVariant]):
            ended_at (datetime.datetime | Unset):
            leader (str | Unset):
            outcome (SignupFormABResultOutcome | Unset):
            runner_up (str | Unset):
            started_on (datetime.date | Unset): The first day counted, in the workspace's time zone
            winner (str | Unset):
     """

    auto_end: bool
    confidence: float
    double_opt_in: bool
    fewest_views: int
    lift: float
    min_views: int
    status: SignupFormABResultStatus
    target_confidence: float
    variants: list[SignupFormABVariant]
    ended_at: datetime.datetime | Unset = UNSET
    leader: str | Unset = UNSET
    outcome: SignupFormABResultOutcome | Unset = UNSET
    runner_up: str | Unset = UNSET
    started_on: datetime.date | Unset = UNSET
    winner: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_ab_variant import SignupFormABVariant # noqa: PLC0415
        auto_end = self.auto_end

        confidence = self.confidence

        double_opt_in = self.double_opt_in

        fewest_views = self.fewest_views

        lift = self.lift

        min_views = self.min_views

        status = self.status.value

        target_confidence = self.target_confidence

        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()

        leader = self.leader

        outcome: str | Unset = UNSET
        if not isinstance(self.outcome, Unset):
            outcome = self.outcome.value


        runner_up = self.runner_up

        started_on: str | Unset = UNSET
        if not isinstance(self.started_on, Unset):
            started_on = self.started_on.isoformat()

        winner = self.winner


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "autoEnd": auto_end,
            "confidence": confidence,
            "doubleOptIn": double_opt_in,
            "fewestViews": fewest_views,
            "lift": lift,
            "minViews": min_views,
            "status": status,
            "targetConfidence": target_confidence,
            "variants": variants,
        })
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at
        if leader is not UNSET:
            field_dict["leader"] = leader
        if outcome is not UNSET:
            field_dict["outcome"] = outcome
        if runner_up is not UNSET:
            field_dict["runnerUp"] = runner_up
        if started_on is not UNSET:
            field_dict["startedOn"] = started_on
        if winner is not UNSET:
            field_dict["winner"] = winner

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_ab_variant import SignupFormABVariant # noqa: PLC0415
        d = dict(src_dict)
        auto_end = d.pop("autoEnd")

        confidence = d.pop("confidence")

        double_opt_in = d.pop("doubleOptIn")

        fewest_views = d.pop("fewestViews")

        lift = d.pop("lift")

        min_views = d.pop("minViews")

        status = SignupFormABResultStatus(d.pop("status"))




        target_confidence = d.pop("targetConfidence")

        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = SignupFormABVariant.from_dict(variants_item_data)



            variants.append(variants_item)


        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        leader = d.pop("leader", UNSET)

        _outcome = d.pop("outcome", UNSET)
        outcome: SignupFormABResultOutcome | Unset
        if isinstance(_outcome,  Unset):
            outcome = UNSET
        else:
            outcome = SignupFormABResultOutcome(_outcome)




        runner_up = d.pop("runnerUp", UNSET)

        _started_on = d.pop("startedOn", UNSET)
        started_on: datetime.date | Unset
        if isinstance(_started_on,  Unset):
            started_on = UNSET
        else:
            started_on = datetime.date.fromisoformat(_started_on)




        winner = d.pop("winner", UNSET)

        signup_form_ab_result = cls(
            auto_end=auto_end,
            confidence=confidence,
            double_opt_in=double_opt_in,
            fewest_views=fewest_views,
            lift=lift,
            min_views=min_views,
            status=status,
            target_confidence=target_confidence,
            variants=variants,
            ended_at=ended_at,
            leader=leader,
            outcome=outcome,
            runner_up=runner_up,
            started_on=started_on,
            winner=winner,
        )

        return signup_form_ab_result

