from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.warmup_hold_alternative import WarmupHoldAlternative





T = TypeVar("T", bound="WarmupCapHold")



@_attrs_define
class WarmupCapHold:
    """ 
        Attributes:
            alternatives (list[WarmupHoldAlternative]):
            campaign (str):
            campaign_id (str):
            held (int):
            paused (bool):
            decision (str | Unset):
            resume_at (datetime.datetime | Unset):
     """

    alternatives: list[WarmupHoldAlternative]
    campaign: str
    campaign_id: str
    held: int
    paused: bool
    decision: str | Unset = UNSET
    resume_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.warmup_hold_alternative import WarmupHoldAlternative # noqa: PLC0415
        alternatives = []
        for alternatives_item_data in self.alternatives:
            alternatives_item = alternatives_item_data.to_dict()
            alternatives.append(alternatives_item)



        campaign = self.campaign

        campaign_id = self.campaign_id

        held = self.held

        paused = self.paused

        decision = self.decision

        resume_at: str | Unset = UNSET
        if not isinstance(self.resume_at, Unset):
            resume_at = self.resume_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alternatives": alternatives,
            "campaign": campaign,
            "campaignId": campaign_id,
            "held": held,
            "paused": paused,
        })
        if decision is not UNSET:
            field_dict["decision"] = decision
        if resume_at is not UNSET:
            field_dict["resumeAt"] = resume_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.warmup_hold_alternative import WarmupHoldAlternative # noqa: PLC0415
        d = dict(src_dict)
        alternatives = []
        _alternatives = d.pop("alternatives")
        for alternatives_item_data in (_alternatives):
            alternatives_item = WarmupHoldAlternative.from_dict(alternatives_item_data)



            alternatives.append(alternatives_item)


        campaign = d.pop("campaign")

        campaign_id = d.pop("campaignId")

        held = d.pop("held")

        paused = d.pop("paused")

        decision = d.pop("decision", UNSET)

        _resume_at = d.pop("resumeAt", UNSET)
        resume_at: datetime.datetime | Unset
        if isinstance(_resume_at,  Unset):
            resume_at = UNSET
        else:
            resume_at = datetime.datetime.fromisoformat(_resume_at)




        warmup_cap_hold = cls(
            alternatives=alternatives,
            campaign=campaign,
            campaign_id=campaign_id,
            held=held,
            paused=paused,
            decision=decision,
            resume_at=resume_at,
        )

        return warmup_cap_hold

