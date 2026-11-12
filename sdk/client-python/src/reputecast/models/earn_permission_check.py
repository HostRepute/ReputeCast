from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_permission_check_access import EarnPermissionCheckAccess
from ..types import UNSET, Unset






T = TypeVar("T", bound="EarnPermissionCheck")



@_attrs_define
class EarnPermissionCheck:
    """ 
        Attributes:
            access (EarnPermissionCheckAccess):
            resource (str):
            ok (bool | Unset):
     """

    access: EarnPermissionCheckAccess
    resource: str
    ok: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        access = self.access.value

        resource = self.resource

        ok = self.ok


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "resource": resource,
        })
        if ok is not UNSET:
            field_dict["ok"] = ok

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        access = EarnPermissionCheckAccess(d.pop("access"))




        resource = d.pop("resource")

        ok = d.pop("ok", UNSET)

        earn_permission_check = cls(
            access=access,
            resource=resource,
            ok=ok,
        )

        return earn_permission_check

