from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.contact_saved_view import ContactSavedView





T = TypeVar("T", bound="ContactSavedViews")



@_attrs_define
class ContactSavedViews:
    """ 
        Attributes:
            all_ (int):
            items (list[ContactSavedView]):
     """

    all_: int
    items: list[ContactSavedView]





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_saved_view import ContactSavedView # noqa: PLC0415
        all_ = self.all_

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_saved_view import ContactSavedView # noqa: PLC0415
        d = dict(src_dict)
        all_ = d.pop("all")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ContactSavedView.from_dict(items_item_data)



            items.append(items_item)


        contact_saved_views = cls(
            all_=all_,
            items=items,
        )

        return contact_saved_views

