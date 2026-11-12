from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_trigger_store import CommerceTriggerStore





T = TypeVar("T", bound="CommerceAutomationOptions")



@_attrs_define
class CommerceAutomationOptions:
    """ 
        Attributes:
            stores (list[CommerceTriggerStore]):
     """

    stores: list[CommerceTriggerStore]





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_trigger_store import CommerceTriggerStore # noqa: PLC0415
        stores = []
        for stores_item_data in self.stores:
            stores_item = stores_item_data.to_dict()
            stores.append(stores_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "stores": stores,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_trigger_store import CommerceTriggerStore # noqa: PLC0415
        d = dict(src_dict)
        stores = []
        _stores = d.pop("stores")
        for stores_item_data in (_stores):
            stores_item = CommerceTriggerStore.from_dict(stores_item_data)



            stores.append(stores_item)


        commerce_automation_options = cls(
            stores=stores,
        )

        return commerce_automation_options

