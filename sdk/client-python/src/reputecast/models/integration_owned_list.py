from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.integration_owned import IntegrationOwned





T = TypeVar("T", bound="IntegrationOwnedList")



@_attrs_define
class IntegrationOwnedList:
    """ 
        Attributes:
            can_manage (bool):
            items (list[IntegrationOwned]):
     """

    can_manage: bool
    items: list[IntegrationOwned]





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_owned import IntegrationOwned # noqa: PLC0415
        can_manage = self.can_manage

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_owned import IntegrationOwned # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = IntegrationOwned.from_dict(items_item_data)



            items.append(items_item)


        integration_owned_list = cls(
            can_manage=can_manage,
            items=items,
        )

        return integration_owned_list

