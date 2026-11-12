from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceProduct")



@_attrs_define
class CommerceProduct:
    """ 
        Attributes:
            available (bool):
            compare_at (int):
            currency (str):
            id (str):
            image_alt (str):
            image_url (str):
            price (int):
            removed (bool):
            title (str):
            url (str):
            inventory (int | Unset):
            removed_at (datetime.datetime | Unset):
     """

    available: bool
    compare_at: int
    currency: str
    id: str
    image_alt: str
    image_url: str
    price: int
    removed: bool
    title: str
    url: str
    inventory: int | Unset = UNSET
    removed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        compare_at = self.compare_at

        currency = self.currency

        id = self.id

        image_alt = self.image_alt

        image_url = self.image_url

        price = self.price

        removed = self.removed

        title = self.title

        url = self.url

        inventory = self.inventory

        removed_at: str | Unset = UNSET
        if not isinstance(self.removed_at, Unset):
            removed_at = self.removed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "compareAt": compare_at,
            "currency": currency,
            "id": id,
            "imageAlt": image_alt,
            "imageUrl": image_url,
            "price": price,
            "removed": removed,
            "title": title,
            "url": url,
        })
        if inventory is not UNSET:
            field_dict["inventory"] = inventory
        if removed_at is not UNSET:
            field_dict["removedAt"] = removed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        compare_at = d.pop("compareAt")

        currency = d.pop("currency")

        id = d.pop("id")

        image_alt = d.pop("imageAlt")

        image_url = d.pop("imageUrl")

        price = d.pop("price")

        removed = d.pop("removed")

        title = d.pop("title")

        url = d.pop("url")

        inventory = d.pop("inventory", UNSET)

        _removed_at = d.pop("removedAt", UNSET)
        removed_at: datetime.datetime | Unset
        if isinstance(_removed_at,  Unset):
            removed_at = UNSET
        else:
            removed_at = datetime.datetime.fromisoformat(_removed_at)




        commerce_product = cls(
            available=available,
            compare_at=compare_at,
            currency=currency,
            id=id,
            image_alt=image_alt,
            image_url=image_url,
            price=price,
            removed=removed,
            title=title,
            url=url,
            inventory=inventory,
            removed_at=removed_at,
        )

        return commerce_product

