from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.list_health_row_health_tone import ListHealthRowHealthTone
from ..models.list_health_row_kind import ListHealthRowKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.audience_tier_counts import AudienceTierCounts





T = TypeVar("T", bound="ListHealthRow")



@_attrs_define
class ListHealthRow:
    """ 
        Attributes:
            complaint_high (bool):
            contacts (int):
            health (str): Excellent, Good, Needs attention, Re-permission needed
            health_tone (ListHealthRowHealthTone):
            id (str):
            kind (ListHealthRowKind):
            name (str):
            net_growth (int):
            tiers (AudienceTierCounts):
            bounce_rate (float | Unset):
            complaint_rate (float | Unset):
            human_opens (float | Unset):
            unsub_rate (float | Unset):
     """

    complaint_high: bool
    contacts: int
    health: str
    health_tone: ListHealthRowHealthTone
    id: str
    kind: ListHealthRowKind
    name: str
    net_growth: int
    tiers: AudienceTierCounts
    bounce_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    human_opens: float | Unset = UNSET
    unsub_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audience_tier_counts import AudienceTierCounts # noqa: PLC0415
        complaint_high = self.complaint_high

        contacts = self.contacts

        health = self.health

        health_tone = self.health_tone.value

        id = self.id

        kind = self.kind.value

        name = self.name

        net_growth = self.net_growth

        tiers = self.tiers.to_dict()

        bounce_rate = self.bounce_rate

        complaint_rate = self.complaint_rate

        human_opens = self.human_opens

        unsub_rate = self.unsub_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "complaintHigh": complaint_high,
            "contacts": contacts,
            "health": health,
            "healthTone": health_tone,
            "id": id,
            "kind": kind,
            "name": name,
            "netGrowth": net_growth,
            "tiers": tiers,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens
        if unsub_rate is not UNSET:
            field_dict["unsubRate"] = unsub_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audience_tier_counts import AudienceTierCounts # noqa: PLC0415
        d = dict(src_dict)
        complaint_high = d.pop("complaintHigh")

        contacts = d.pop("contacts")

        health = d.pop("health")

        health_tone = ListHealthRowHealthTone(d.pop("healthTone"))




        id = d.pop("id")

        kind = ListHealthRowKind(d.pop("kind"))




        name = d.pop("name")

        net_growth = d.pop("netGrowth")

        tiers = AudienceTierCounts.from_dict(d.pop("tiers"))




        bounce_rate = d.pop("bounceRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        human_opens = d.pop("humanOpens", UNSET)

        unsub_rate = d.pop("unsubRate", UNSET)

        list_health_row = cls(
            complaint_high=complaint_high,
            contacts=contacts,
            health=health,
            health_tone=health_tone,
            id=id,
            kind=kind,
            name=name,
            net_growth=net_growth,
            tiers=tiers,
            bounce_rate=bounce_rate,
            complaint_rate=complaint_rate,
            human_opens=human_opens,
            unsub_rate=unsub_rate,
        )

        return list_health_row

