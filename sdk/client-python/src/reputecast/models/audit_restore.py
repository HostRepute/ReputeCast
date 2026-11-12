from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audit_restore_kind import AuditRestoreKind
from typing import cast
import datetime






T = TypeVar("T", bound="AuditRestore")



@_attrs_define
class AuditRestore:
    """ 
        Attributes:
            id (str):
            kind (AuditRestoreKind):
            name (str):
            until (datetime.datetime):
     """

    id: str
    kind: AuditRestoreKind
    name: str
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind.value

        name = self.name

        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "name": name,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = AuditRestoreKind(d.pop("kind"))




        name = d.pop("name")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        audit_restore = cls(
            id=id,
            kind=kind,
            name=name,
            until=until,
        )

        return audit_restore

