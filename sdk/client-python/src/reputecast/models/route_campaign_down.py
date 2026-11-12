from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_campaign_down_reason import RouteCampaignDownReason
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="RouteCampaignDown")



@_attrs_define
class RouteCampaignDown:
    """ 
        Attributes:
            checks (list[str]):
            held (int):
            moved (int):
            status (str):
            reason (RouteCampaignDownReason | Unset):
            to (str | Unset):
     """

    checks: list[str]
    held: int
    moved: int
    status: str
    reason: RouteCampaignDownReason | Unset = UNSET
    to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checks = self.checks



        held = self.held

        moved = self.moved

        status = self.status

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checks": checks,
            "held": held,
            "moved": moved,
            "status": status,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checks = cast(list[str], d.pop("checks"))


        held = d.pop("held")

        moved = d.pop("moved")

        status = d.pop("status")

        _reason = d.pop("reason", UNSET)
        reason: RouteCampaignDownReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = RouteCampaignDownReason(_reason)




        to = d.pop("to", UNSET)

        route_campaign_down = cls(
            checks=checks,
            held=held,
            moved=moved,
            status=status,
            reason=reason,
            to=to,
        )

        return route_campaign_down

