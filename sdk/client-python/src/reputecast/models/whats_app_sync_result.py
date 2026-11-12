from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppSyncResult")



@_attrs_define
class WhatsAppSyncResult:
    """ 
        Attributes:
            added (list[str]):
            changed (list[str]):
            removed (list[str]):
            synced_at (datetime.datetime):
            templates (int):
     """

    added: list[str]
    changed: list[str]
    removed: list[str]
    synced_at: datetime.datetime
    templates: int





    def to_dict(self) -> dict[str, Any]:
        added = self.added



        changed = self.changed



        removed = self.removed



        synced_at = self.synced_at.isoformat()

        templates = self.templates


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "changed": changed,
            "removed": removed,
            "syncedAt": synced_at,
            "templates": templates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        added = cast(list[str], d.pop("added"))


        changed = cast(list[str], d.pop("changed"))


        removed = cast(list[str], d.pop("removed"))


        synced_at = datetime.datetime.fromisoformat(d.pop("syncedAt"))




        templates = d.pop("templates")

        whats_app_sync_result = cls(
            added=added,
            changed=changed,
            removed=removed,
            synced_at=synced_at,
            templates=templates,
        )

        return whats_app_sync_result

