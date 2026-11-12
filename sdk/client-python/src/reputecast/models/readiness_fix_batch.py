from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReadinessFixBatch")



@_attrs_define
class ReadinessFixBatch:
    """ 
        Attributes:
            applied (list[str]):
            applied_at (datetime.datetime):
            applied_by (str):
            count (int):
            id (str):
            undoable (bool):
            route (bool | Unset):
            version (int | Unset):
     """

    applied: list[str]
    applied_at: datetime.datetime
    applied_by: str
    count: int
    id: str
    undoable: bool
    route: bool | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        applied = self.applied



        applied_at = self.applied_at.isoformat()

        applied_by = self.applied_by

        count = self.count

        id = self.id

        undoable = self.undoable

        route = self.route

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "applied": applied,
            "appliedAt": applied_at,
            "appliedBy": applied_by,
            "count": count,
            "id": id,
            "undoable": undoable,
        })
        if route is not UNSET:
            field_dict["route"] = route
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        applied = cast(list[str], d.pop("applied"))


        applied_at = datetime.datetime.fromisoformat(d.pop("appliedAt"))




        applied_by = d.pop("appliedBy")

        count = d.pop("count")

        id = d.pop("id")

        undoable = d.pop("undoable")

        route = d.pop("route", UNSET)

        version = d.pop("version", UNSET)

        readiness_fix_batch = cls(
            applied=applied,
            applied_at=applied_at,
            applied_by=applied_by,
            count=count,
            id=id,
            undoable=undoable,
            route=route,
            version=version,
        )

        return readiness_fix_batch

