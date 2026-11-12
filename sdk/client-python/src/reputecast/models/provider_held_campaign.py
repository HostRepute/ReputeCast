from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.provider_route_share import ProviderRouteShare





T = TypeVar("T", bound="ProviderHeldCampaign")



@_attrs_define
class ProviderHeldCampaign:
    """ 
        Attributes:
            held (int):
            id (str):
            name (str):
            other_shares (list[ProviderRouteShare]):
            planned (int):
            sent (int):
            decision (str | Unset): wait, move or cancel once someone decided
     """

    held: int
    id: str
    name: str
    other_shares: list[ProviderRouteShare]
    planned: int
    sent: int
    decision: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_route_share import ProviderRouteShare # noqa: PLC0415
        held = self.held

        id = self.id

        name = self.name

        other_shares = []
        for other_shares_item_data in self.other_shares:
            other_shares_item = other_shares_item_data.to_dict()
            other_shares.append(other_shares_item)



        planned = self.planned

        sent = self.sent

        decision = self.decision


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "held": held,
            "id": id,
            "name": name,
            "otherShares": other_shares,
            "planned": planned,
            "sent": sent,
        })
        if decision is not UNSET:
            field_dict["decision"] = decision

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_route_share import ProviderRouteShare # noqa: PLC0415
        d = dict(src_dict)
        held = d.pop("held")

        id = d.pop("id")

        name = d.pop("name")

        other_shares = []
        _other_shares = d.pop("otherShares")
        for other_shares_item_data in (_other_shares):
            other_shares_item = ProviderRouteShare.from_dict(other_shares_item_data)



            other_shares.append(other_shares_item)


        planned = d.pop("planned")

        sent = d.pop("sent")

        decision = d.pop("decision", UNSET)

        provider_held_campaign = cls(
            held=held,
            id=id,
            name=name,
            other_shares=other_shares,
            planned=planned,
            sent=sent,
            decision=decision,
        )

        return provider_held_campaign

