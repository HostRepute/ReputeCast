from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_warmup_hold_decision import LiveWarmupHoldDecision
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.live_warmup_target import LiveWarmupTarget





T = TypeVar("T", bound="LiveWarmupHold")



@_attrs_define
class LiveWarmupHold:
    """ 
        Attributes:
            cap (int):
            held (int):
            paused (bool):
            plan_id (str):
            targets (list[LiveWarmupTarget]):
            decision (LiveWarmupHoldDecision | Unset):
            since (datetime.datetime | Unset):
            until (datetime.datetime | Unset):
     """

    cap: int
    held: int
    paused: bool
    plan_id: str
    targets: list[LiveWarmupTarget]
    decision: LiveWarmupHoldDecision | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_warmup_target import LiveWarmupTarget # noqa: PLC0415
        cap = self.cap

        held = self.held

        paused = self.paused

        plan_id = self.plan_id

        targets = []
        for targets_item_data in self.targets:
            targets_item = targets_item_data.to_dict()
            targets.append(targets_item)



        decision: str | Unset = UNSET
        if not isinstance(self.decision, Unset):
            decision = self.decision.value


        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        until: str | Unset = UNSET
        if not isinstance(self.until, Unset):
            until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "held": held,
            "paused": paused,
            "planId": plan_id,
            "targets": targets,
        })
        if decision is not UNSET:
            field_dict["decision"] = decision
        if since is not UNSET:
            field_dict["since"] = since
        if until is not UNSET:
            field_dict["until"] = until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_warmup_target import LiveWarmupTarget # noqa: PLC0415
        d = dict(src_dict)
        cap = d.pop("cap")

        held = d.pop("held")

        paused = d.pop("paused")

        plan_id = d.pop("planId")

        targets = []
        _targets = d.pop("targets")
        for targets_item_data in (_targets):
            targets_item = LiveWarmupTarget.from_dict(targets_item_data)



            targets.append(targets_item)


        _decision = d.pop("decision", UNSET)
        decision: LiveWarmupHoldDecision | Unset
        if isinstance(_decision,  Unset):
            decision = UNSET
        else:
            decision = LiveWarmupHoldDecision(_decision)




        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        _until = d.pop("until", UNSET)
        until: datetime.datetime | Unset
        if isinstance(_until,  Unset):
            until = UNSET
        else:
            until = datetime.datetime.fromisoformat(_until)




        live_warmup_hold = cls(
            cap=cap,
            held=held,
            paused=paused,
            plan_id=plan_id,
            targets=targets,
            decision=decision,
            since=since,
            until=until,
        )

        return live_warmup_hold

