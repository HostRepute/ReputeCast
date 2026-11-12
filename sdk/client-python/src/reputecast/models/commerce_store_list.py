from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_store_summary import CommerceStoreSummary





T = TypeVar("T", bound="CommerceStoreList")



@_attrs_define
class CommerceStoreList:
    """ 
        Attributes:
            can_manage (bool):
            helpers (list[str]):
            stores (list[CommerceStoreSummary]):
     """

    can_manage: bool
    helpers: list[str]
    stores: list[CommerceStoreSummary]





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_store_summary import CommerceStoreSummary # noqa: PLC0415
        can_manage = self.can_manage

        helpers = self.helpers



        stores = []
        for stores_item_data in self.stores:
            stores_item = stores_item_data.to_dict()
            stores.append(stores_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "helpers": helpers,
            "stores": stores,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_store_summary import CommerceStoreSummary # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        stores = []
        _stores = d.pop("stores")
        for stores_item_data in (_stores):
            stores_item = CommerceStoreSummary.from_dict(stores_item_data)



            stores.append(stores_item)


        commerce_store_list = cls(
            can_manage=can_manage,
            helpers=helpers,
            stores=stores,
        )

        return commerce_store_list

