from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="RoleImpactInput")



@_attrs_define
class RoleImpactInput:
    """ 
        Attributes:
            permissions (list[str]):
            role_id (str | Unset):
     """

    permissions: list[str]
    role_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        permissions = self.permissions



        role_id = self.role_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "permissions": permissions,
        })
        if role_id is not UNSET:
            field_dict["roleId"] = role_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        permissions = cast(list[str], d.pop("permissions"))


        role_id = d.pop("roleId", UNSET)

        role_impact_input = cls(
            permissions=permissions,
            role_id=role_id,
        )

        return role_impact_input

