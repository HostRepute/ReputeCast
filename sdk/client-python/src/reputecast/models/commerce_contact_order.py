from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_contact_order_credited_kind import CommerceContactOrderCreditedKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceContactOrder")



@_attrs_define
class CommerceContactOrder:
    """ 
        Attributes:
            accepted (bool):
            currency (str):
            id (str):
            items (list[str]):
            name (str):
            placed_at (datetime.datetime):
            platform (str):
            refunded (int):
            store (str):
            store_id (str):
            total (int):
            credited (str | Unset):
            credited_kind (CommerceContactOrderCreditedKind | Unset):
     """

    accepted: bool
    currency: str
    id: str
    items: list[str]
    name: str
    placed_at: datetime.datetime
    platform: str
    refunded: int
    store: str
    store_id: str
    total: int
    credited: str | Unset = UNSET
    credited_kind: CommerceContactOrderCreditedKind | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        currency = self.currency

        id = self.id

        items = self.items



        name = self.name

        placed_at = self.placed_at.isoformat()

        platform = self.platform

        refunded = self.refunded

        store = self.store

        store_id = self.store_id

        total = self.total

        credited = self.credited

        credited_kind: str | Unset = UNSET
        if not isinstance(self.credited_kind, Unset):
            credited_kind = self.credited_kind.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "currency": currency,
            "id": id,
            "items": items,
            "name": name,
            "placedAt": placed_at,
            "platform": platform,
            "refunded": refunded,
            "store": store,
            "storeId": store_id,
            "total": total,
        })
        if credited is not UNSET:
            field_dict["credited"] = credited
        if credited_kind is not UNSET:
            field_dict["creditedKind"] = credited_kind

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        currency = d.pop("currency")

        id = d.pop("id")

        items = cast(list[str], d.pop("items"))


        name = d.pop("name")

        placed_at = datetime.datetime.fromisoformat(d.pop("placedAt"))




        platform = d.pop("platform")

        refunded = d.pop("refunded")

        store = d.pop("store")

        store_id = d.pop("storeId")

        total = d.pop("total")

        credited = d.pop("credited", UNSET)

        _credited_kind = d.pop("creditedKind", UNSET)
        credited_kind: CommerceContactOrderCreditedKind | Unset
        if isinstance(_credited_kind,  Unset):
            credited_kind = UNSET
        else:
            credited_kind = CommerceContactOrderCreditedKind(_credited_kind)




        commerce_contact_order = cls(
            accepted=accepted,
            currency=currency,
            id=id,
            items=items,
            name=name,
            placed_at=placed_at,
            platform=platform,
            refunded=refunded,
            store=store,
            store_id=store_id,
            total=total,
            credited=credited,
            credited_kind=credited_kind,
        )

        return commerce_contact_order

