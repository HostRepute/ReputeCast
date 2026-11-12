from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.api_key_locked_permission import APIKeyLockedPermission
  from ..models.api_key_view import APIKeyView
  from ..models.permission_info import PermissionInfo
  from ..models.scoped_list import ScopedList





T = TypeVar("T", bound="APIKeyList")



@_attrs_define
class APIKeyList:
    """ 
        Attributes:
            endpoint (str):
            groups (list[str]):
            held (list[str]):
            items (list[APIKeyView]):
            lists (list[ScopedList]):
            locked (list[APIKeyLockedPermission]):
            permissions (list[PermissionInfo]):
            proxy_warning (bool): Requests come through a proxy that http.trusted_proxies doesn't list, so the addresses
                keys are called from may be the proxy's
            your_ip (str): The address of this request as ReputeCast resolved it; empty when unknown
     """

    endpoint: str
    groups: list[str]
    held: list[str]
    items: list[APIKeyView]
    lists: list[ScopedList]
    locked: list[APIKeyLockedPermission]
    permissions: list[PermissionInfo]
    proxy_warning: bool
    your_ip: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.api_key_locked_permission import APIKeyLockedPermission # noqa: PLC0415
        from ..models.api_key_view import APIKeyView # noqa: PLC0415
        from ..models.permission_info import PermissionInfo # noqa: PLC0415
        from ..models.scoped_list import ScopedList # noqa: PLC0415
        endpoint = self.endpoint

        groups = self.groups



        held = self.held



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        locked = []
        for locked_item_data in self.locked:
            locked_item = locked_item_data.to_dict()
            locked.append(locked_item)



        permissions = []
        for permissions_item_data in self.permissions:
            permissions_item = permissions_item_data.to_dict()
            permissions.append(permissions_item)



        proxy_warning = self.proxy_warning

        your_ip = self.your_ip


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "endpoint": endpoint,
            "groups": groups,
            "held": held,
            "items": items,
            "lists": lists,
            "locked": locked,
            "permissions": permissions,
            "proxyWarning": proxy_warning,
            "yourIp": your_ip,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.api_key_locked_permission import APIKeyLockedPermission # noqa: PLC0415
        from ..models.api_key_view import APIKeyView # noqa: PLC0415
        from ..models.permission_info import PermissionInfo # noqa: PLC0415
        from ..models.scoped_list import ScopedList # noqa: PLC0415
        d = dict(src_dict)
        endpoint = d.pop("endpoint")

        groups = cast(list[str], d.pop("groups"))


        held = cast(list[str], d.pop("held"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = APIKeyView.from_dict(items_item_data)



            items.append(items_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ScopedList.from_dict(lists_item_data)



            lists.append(lists_item)


        locked = []
        _locked = d.pop("locked")
        for locked_item_data in (_locked):
            locked_item = APIKeyLockedPermission.from_dict(locked_item_data)



            locked.append(locked_item)


        permissions = []
        _permissions = d.pop("permissions")
        for permissions_item_data in (_permissions):
            permissions_item = PermissionInfo.from_dict(permissions_item_data)



            permissions.append(permissions_item)


        proxy_warning = d.pop("proxyWarning")

        your_ip = d.pop("yourIp")

        api_key_list = cls(
            endpoint=endpoint,
            groups=groups,
            held=held,
            items=items,
            lists=lists,
            locked=locked,
            permissions=permissions,
            proxy_warning=proxy_warning,
            your_ip=your_ip,
        )

        return api_key_list

