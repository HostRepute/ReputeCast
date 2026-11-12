from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_content_option_kind import PlacementContentOptionKind
from typing import cast
import datetime






T = TypeVar("T", bound="PlacementContentOption")



@_attrs_define
class PlacementContentOption:
    """ 
        Attributes:
            at (datetime.datetime):
            id (str):
            kind (PlacementContentOptionKind):
            name (str):
            note (str): A campaign's status, template, a transactional template's key, or a digest's state (on, paused, off)
     """

    at: datetime.datetime
    id: str
    kind: PlacementContentOptionKind
    name: str
    note: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        kind = self.kind.value

        name = self.name

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "id": id,
            "kind": kind,
            "name": name,
            "note": note,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        kind = PlacementContentOptionKind(d.pop("kind"))




        name = d.pop("name")

        note = d.pop("note")

        placement_content_option = cls(
            at=at,
            id=id,
            kind=kind,
            name=name,
            note=note,
        )

        return placement_content_option

