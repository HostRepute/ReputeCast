from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.synced_uses import SyncedUses





T = TypeVar("T", bound="SyncedBlock")



@_attrs_define
class SyncedBlock:
    """ 
        Attributes:
            kind (str): header, footer, or a named block's key (R4.A3)
            mjml (str): One section or wrapper, marked rc-synced
            name (str):
            updated_at (datetime.datetime):
            updated_by (str):
            uses (SyncedUses):
            version (int):
     """

    kind: str
    mjml: str
    name: str
    updated_at: datetime.datetime
    updated_by: str
    uses: SyncedUses
    version: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.synced_uses import SyncedUses # noqa: PLC0415
        kind = self.kind

        mjml = self.mjml

        name = self.name

        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        uses = self.uses.to_dict()

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "mjml": mjml,
            "name": name,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
            "uses": uses,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.synced_uses import SyncedUses # noqa: PLC0415
        d = dict(src_dict)
        kind = d.pop("kind")

        mjml = d.pop("mjml")

        name = d.pop("name")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        uses = SyncedUses.from_dict(d.pop("uses"))




        version = d.pop("version")

        synced_block = cls(
            kind=kind,
            mjml=mjml,
            name=name,
            updated_at=updated_at,
            updated_by=updated_by,
            uses=uses,
            version=version,
        )

        return synced_block

