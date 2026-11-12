from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.tracker_site import TrackerSite
  from ..models.tracker_store_option import TrackerStoreOption





T = TypeVar("T", bound="TrackerSiteList")



@_attrs_define
class TrackerSiteList:
    """ 
        Attributes:
            can_manage (bool):
            helpers (list[str]):
            items (list[TrackerSite]):
            script_url (str):
            stores (list[TrackerStoreOption]):
     """

    can_manage: bool
    helpers: list[str]
    items: list[TrackerSite]
    script_url: str
    stores: list[TrackerStoreOption]





    def to_dict(self) -> dict[str, Any]:
        from ..models.tracker_site import TrackerSite # noqa: PLC0415
        from ..models.tracker_store_option import TrackerStoreOption # noqa: PLC0415
        can_manage = self.can_manage

        helpers = self.helpers



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        script_url = self.script_url

        stores = []
        for stores_item_data in self.stores:
            stores_item = stores_item_data.to_dict()
            stores.append(stores_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "helpers": helpers,
            "items": items,
            "scriptUrl": script_url,
            "stores": stores,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tracker_site import TrackerSite # noqa: PLC0415
        from ..models.tracker_store_option import TrackerStoreOption # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TrackerSite.from_dict(items_item_data)



            items.append(items_item)


        script_url = d.pop("scriptUrl")

        stores = []
        _stores = d.pop("stores")
        for stores_item_data in (_stores):
            stores_item = TrackerStoreOption.from_dict(stores_item_data)



            stores.append(stores_item)


        tracker_site_list = cls(
            can_manage=can_manage,
            helpers=helpers,
            items=items,
            script_url=script_url,
            stores=stores,
        )

        return tracker_site_list

