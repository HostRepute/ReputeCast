from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_activity_store_platform import ContactActivityStorePlatform
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ContactActivityStore")



@_attrs_define
class ContactActivityStore:
    """ 
        Attributes:
            accepted (bool):
            currency (str):
            items (list[str]):
            name (str):
            platform (ContactActivityStorePlatform):
            total (int):
            credited (str | Unset):
            flow (str | Unset):
            order (str | Unset):
     """

    accepted: bool
    currency: str
    items: list[str]
    name: str
    platform: ContactActivityStorePlatform
    total: int
    credited: str | Unset = UNSET
    flow: str | Unset = UNSET
    order: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        currency = self.currency

        items = self.items



        name = self.name

        platform = self.platform.value

        total = self.total

        credited = self.credited

        flow = self.flow

        order = self.order


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "currency": currency,
            "items": items,
            "name": name,
            "platform": platform,
            "total": total,
        })
        if credited is not UNSET:
            field_dict["credited"] = credited
        if flow is not UNSET:
            field_dict["flow"] = flow
        if order is not UNSET:
            field_dict["order"] = order

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        currency = d.pop("currency")

        items = cast(list[str], d.pop("items"))


        name = d.pop("name")

        platform = ContactActivityStorePlatform(d.pop("platform"))




        total = d.pop("total")

        credited = d.pop("credited", UNSET)

        flow = d.pop("flow", UNSET)

        order = d.pop("order", UNSET)

        contact_activity_store = cls(
            accepted=accepted,
            currency=currency,
            items=items,
            name=name,
            platform=platform,
            total=total,
            credited=credited,
            flow=flow,
            order=order,
        )

        return contact_activity_store

