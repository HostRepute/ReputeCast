from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.outside_kind import OutsideKind






T = TypeVar("T", bound="Outside")



@_attrs_define
class Outside:
    """ 
        Attributes:
            char (str):
            count (int):
            first (int):
            kind (OutsideKind):
            removable (bool):
            replacement (str):
     """

    char: str
    count: int
    first: int
    kind: OutsideKind
    removable: bool
    replacement: str





    def to_dict(self) -> dict[str, Any]:
        char = self.char

        count = self.count

        first = self.first

        kind = self.kind.value

        removable = self.removable

        replacement = self.replacement


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "char": char,
            "count": count,
            "first": first,
            "kind": kind,
            "removable": removable,
            "replacement": replacement,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        char = d.pop("char")

        count = d.pop("count")

        first = d.pop("first")

        kind = OutsideKind(d.pop("kind"))




        removable = d.pop("removable")

        replacement = d.pop("replacement")

        outside = cls(
            char=char,
            count=count,
            first=first,
            kind=kind,
            removable=removable,
            replacement=replacement,
        )

        return outside

