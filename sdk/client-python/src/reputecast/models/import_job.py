from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_job_kind import ImportJobKind
from ..models.import_job_status import ImportJobStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ImportJob")



@_attrs_define
class ImportJob:
    """ 
        Attributes:
            href (str):
            id (str):
            kind (ImportJobKind):
            status (ImportJobStatus):
            title (str):
            detail (str | Unset):
            download_url (str | Unset):
            eta (str | Unset):
            finished_at (datetime.datetime | Unset):
            progress (int | Unset):
            started_at (datetime.datetime | Unset):
     """

    href: str
    id: str
    kind: ImportJobKind
    status: ImportJobStatus
    title: str
    detail: str | Unset = UNSET
    download_url: str | Unset = UNSET
    eta: str | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    progress: int | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        href = self.href

        id = self.id

        kind = self.kind.value

        status = self.status.value

        title = self.title

        detail = self.detail

        download_url = self.download_url

        eta = self.eta

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        progress = self.progress

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "href": href,
            "id": id,
            "kind": kind,
            "status": status,
            "title": title,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if download_url is not UNSET:
            field_dict["downloadUrl"] = download_url
        if eta is not UNSET:
            field_dict["eta"] = eta
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if progress is not UNSET:
            field_dict["progress"] = progress
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        href = d.pop("href")

        id = d.pop("id")

        kind = ImportJobKind(d.pop("kind"))




        status = ImportJobStatus(d.pop("status"))




        title = d.pop("title")

        detail = d.pop("detail", UNSET)

        download_url = d.pop("downloadUrl", UNSET)

        eta = d.pop("eta", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        progress = d.pop("progress", UNSET)

        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        import_job = cls(
            href=href,
            id=id,
            kind=kind,
            status=status,
            title=title,
            detail=detail,
            download_url=download_url,
            eta=eta,
            finished_at=finished_at,
            progress=progress,
            started_at=started_at,
        )

        return import_job

