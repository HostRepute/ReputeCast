from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ImportSummary")



@_attrs_define
class ImportSummary:
    """ 
        Attributes:
            contacts (int): Contacts imported, merged or imported as pending
            created_at (datetime.datetime):
            created_by (str):
            file_name (str):
            id (str):
            source (str):
            source_name (str):
            status (str):
            step (str):
            finished_at (datetime.datetime | Unset):
     """

    contacts: int
    created_at: datetime.datetime
    created_by: str
    file_name: str
    id: str
    source: str
    source_name: str
    status: str
    step: str
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        file_name = self.file_name

        id = self.id

        source = self.source

        source_name = self.source_name

        status = self.status

        step = self.step

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "createdAt": created_at,
            "createdBy": created_by,
            "fileName": file_name,
            "id": id,
            "source": source,
            "sourceName": source_name,
            "status": status,
            "step": step,
        })
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        file_name = d.pop("fileName")

        id = d.pop("id")

        source = d.pop("source")

        source_name = d.pop("sourceName")

        status = d.pop("status")

        step = d.pop("step")

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        import_summary = cls(
            contacts=contacts,
            created_at=created_at,
            created_by=created_by,
            file_name=file_name,
            id=id,
            source=source,
            source_name=source_name,
            status=status,
            step=step,
            finished_at=finished_at,
        )

        return import_summary

