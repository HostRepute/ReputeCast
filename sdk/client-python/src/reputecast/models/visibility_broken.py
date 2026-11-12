from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.visibility_broken_kind import VisibilityBrokenKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="VisibilityBroken")



@_attrs_define
class VisibilityBroken:
    """ 
        Attributes:
            condition (int):
            kind (VisibilityBrokenKind):
            name (str):
            deleted_at (datetime.datetime | Unset):
            deleted_by (str | Unset):
     """

    condition: int
    kind: VisibilityBrokenKind
    name: str
    deleted_at: datetime.datetime | Unset = UNSET
    deleted_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        condition = self.condition

        kind = self.kind.value

        name = self.name

        deleted_at: str | Unset = UNSET
        if not isinstance(self.deleted_at, Unset):
            deleted_at = self.deleted_at.isoformat()

        deleted_by = self.deleted_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "condition": condition,
            "kind": kind,
            "name": name,
        })
        if deleted_at is not UNSET:
            field_dict["deletedAt"] = deleted_at
        if deleted_by is not UNSET:
            field_dict["deletedBy"] = deleted_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        condition = d.pop("condition")

        kind = VisibilityBrokenKind(d.pop("kind"))




        name = d.pop("name")

        _deleted_at = d.pop("deletedAt", UNSET)
        deleted_at: datetime.datetime | Unset
        if isinstance(_deleted_at,  Unset):
            deleted_at = UNSET
        else:
            deleted_at = datetime.datetime.fromisoformat(_deleted_at)




        deleted_by = d.pop("deletedBy", UNSET)

        visibility_broken = cls(
            condition=condition,
            kind=kind,
            name=name,
            deleted_at=deleted_at,
            deleted_by=deleted_by,
        )

        return visibility_broken

