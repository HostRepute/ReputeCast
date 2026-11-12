from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_policy_on_serious import AccessibilityPolicyOnSerious
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AccessibilityPolicy")



@_attrs_define
class AccessibilityPolicy:
    """ 
        Attributes:
            can_manage (bool):
            helpers (list[str]):
            min_text_px (int):
            on_serious (AccessibilityPolicyOnSerious):
            updated_at (datetime.datetime | Unset):
            updated_by (str | Unset):
     """

    can_manage: bool
    helpers: list[str]
    min_text_px: int
    on_serious: AccessibilityPolicyOnSerious
    updated_at: datetime.datetime | Unset = UNSET
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can_manage = self.can_manage

        helpers = self.helpers



        min_text_px = self.min_text_px

        on_serious = self.on_serious.value

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "helpers": helpers,
            "minTextPx": min_text_px,
            "onSerious": on_serious,
        })
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        min_text_px = d.pop("minTextPx")

        on_serious = AccessibilityPolicyOnSerious(d.pop("onSerious"))




        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        updated_by = d.pop("updatedBy", UNSET)

        accessibility_policy = cls(
            can_manage=can_manage,
            helpers=helpers,
            min_text_px=min_text_px,
            on_serious=on_serious,
            updated_at=updated_at,
            updated_by=updated_by,
        )

        return accessibility_policy

