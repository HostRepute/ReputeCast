from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ApprovalAutomationSnapshot")



@_attrs_define
class ApprovalAutomationSnapshot:
    """ 
        Attributes:
            changes (list[str]):
            migration (str):
            version (int):
     """

    changes: list[str]
    migration: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        changes = self.changes



        migration = self.migration

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "migration": migration,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        changes = cast(list[str], d.pop("changes"))


        migration = d.pop("migration")

        version = d.pop("version")

        approval_automation_snapshot = cls(
            changes=changes,
            migration=migration,
            version=version,
        )

        return approval_automation_snapshot

