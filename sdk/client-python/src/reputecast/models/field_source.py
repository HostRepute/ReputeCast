from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.field_source_kind import FieldSourceKind
from typing import cast
import datetime






T = TypeVar("T", bound="FieldSource")



@_attrs_define
class FieldSource:
    """ 
        Attributes:
            at (datetime.datetime):
            field (str):
            kind (FieldSourceKind):
            name (str):
     """

    at: datetime.datetime
    field: str
    kind: FieldSourceKind
    name: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        field = self.field

        kind = self.kind.value

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "field": field,
            "kind": kind,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        field = d.pop("field")

        kind = FieldSourceKind(d.pop("kind"))




        name = d.pop("name")

        field_source = cls(
            at=at,
            field=field,
            kind=kind,
            name=name,
        )

        return field_source

