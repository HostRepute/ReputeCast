from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.contact_purchase import ContactPurchase





T = TypeVar("T", bound="ContactPurchases")



@_attrs_define
class ContactPurchases:
    """ 
        Attributes:
            can_manage (bool):
            currency (str):
            items (list[ContactPurchase]):
            spent_cents (int):
     """

    can_manage: bool
    currency: str
    items: list[ContactPurchase]
    spent_cents: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_purchase import ContactPurchase # noqa: PLC0415
        can_manage = self.can_manage

        currency = self.currency

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        spent_cents = self.spent_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "currency": currency,
            "items": items,
            "spentCents": spent_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_purchase import ContactPurchase # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        currency = d.pop("currency")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ContactPurchase.from_dict(items_item_data)



            items.append(items_item)


        spent_cents = d.pop("spentCents")

        contact_purchases = cls(
            can_manage=can_manage,
            currency=currency,
            items=items,
            spent_cents=spent_cents,
        )

        return contact_purchases

