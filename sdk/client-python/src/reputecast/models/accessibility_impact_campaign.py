from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_impact_campaign_status import AccessibilityImpactCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AccessibilityImpactCampaign")



@_attrs_define
class AccessibilityImpactCampaign:
    """ 
        Attributes:
            held (bool):
            id (str):
            name (str):
            serious (int):
            status (AccessibilityImpactCampaignStatus):
            summary (str):
            scheduled_at (datetime.datetime | Unset):
     """

    held: bool
    id: str
    name: str
    serious: int
    status: AccessibilityImpactCampaignStatus
    summary: str
    scheduled_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        held = self.held

        id = self.id

        name = self.name

        serious = self.serious

        status = self.status.value

        summary = self.summary

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "held": held,
            "id": id,
            "name": name,
            "serious": serious,
            "status": status,
            "summary": summary,
        })
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        held = d.pop("held")

        id = d.pop("id")

        name = d.pop("name")

        serious = d.pop("serious")

        status = AccessibilityImpactCampaignStatus(d.pop("status"))




        summary = d.pop("summary")

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        accessibility_impact_campaign = cls(
            held=held,
            id=id,
            name=name,
            serious=serious,
            status=status,
            summary=summary,
            scheduled_at=scheduled_at,
        )

        return accessibility_impact_campaign

