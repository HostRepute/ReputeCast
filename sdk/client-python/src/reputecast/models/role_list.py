from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.permission_info import PermissionInfo
  from ..models.role_summary import RoleSummary





T = TypeVar("T", bound="RoleList")



@_attrs_define
class RoleList:
    """ 
        Attributes:
            can_manage (bool):
            groups (list[str]):
            held (list[str]):
            helpers (list[str]):
            permissions (list[PermissionInfo]):
            roles (list[RoleSummary]):
     """

    can_manage: bool
    groups: list[str]
    held: list[str]
    helpers: list[str]
    permissions: list[PermissionInfo]
    roles: list[RoleSummary]





    def to_dict(self) -> dict[str, Any]:
        from ..models.permission_info import PermissionInfo # noqa: PLC0415
        from ..models.role_summary import RoleSummary # noqa: PLC0415
        can_manage = self.can_manage

        groups = self.groups



        held = self.held



        helpers = self.helpers



        permissions = []
        for permissions_item_data in self.permissions:
            permissions_item = permissions_item_data.to_dict()
            permissions.append(permissions_item)



        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "groups": groups,
            "held": held,
            "helpers": helpers,
            "permissions": permissions,
            "roles": roles,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.permission_info import PermissionInfo # noqa: PLC0415
        from ..models.role_summary import RoleSummary # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        groups = cast(list[str], d.pop("groups"))


        held = cast(list[str], d.pop("held"))


        helpers = cast(list[str], d.pop("helpers"))


        permissions = []
        _permissions = d.pop("permissions")
        for permissions_item_data in (_permissions):
            permissions_item = PermissionInfo.from_dict(permissions_item_data)



            permissions.append(permissions_item)


        roles = []
        _roles = d.pop("roles")
        for roles_item_data in (_roles):
            roles_item = RoleSummary.from_dict(roles_item_data)



            roles.append(roles_item)


        role_list = cls(
            can_manage=can_manage,
            groups=groups,
            held=held,
            helpers=helpers,
            permissions=permissions,
            roles=roles,
        )

        return role_list

