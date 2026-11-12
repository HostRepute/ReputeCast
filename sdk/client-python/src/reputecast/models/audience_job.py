from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audience_job_kind import AudienceJobKind
from ..models.audience_job_status import AudienceJobStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.audience_job_result import AudienceJobResult





T = TypeVar("T", bound="AudienceJob")



@_attrs_define
class AudienceJob:
    """ 
        Attributes:
            created_at (datetime.datetime):
            done (int):
            id (str):
            kind (AudienceJobKind):
            result (AudienceJobResult):
            status (AudienceJobStatus):
            total (int):
            download_url (str | Unset):
            error (str | Unset):
            expires_at (datetime.datetime | Unset):
            finished_at (datetime.datetime | Unset):
     """

    created_at: datetime.datetime
    done: int
    id: str
    kind: AudienceJobKind
    result: AudienceJobResult
    status: AudienceJobStatus
    total: int
    download_url: str | Unset = UNSET
    error: str | Unset = UNSET
    expires_at: datetime.datetime | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audience_job_result import AudienceJobResult # noqa: PLC0415
        created_at = self.created_at.isoformat()

        done = self.done

        id = self.id

        kind = self.kind.value

        result = self.result.to_dict()

        status = self.status.value

        total = self.total

        download_url = self.download_url

        error = self.error

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "done": done,
            "id": id,
            "kind": kind,
            "result": result,
            "status": status,
            "total": total,
        })
        if download_url is not UNSET:
            field_dict["downloadUrl"] = download_url
        if error is not UNSET:
            field_dict["error"] = error
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audience_job_result import AudienceJobResult # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        done = d.pop("done")

        id = d.pop("id")

        kind = AudienceJobKind(d.pop("kind"))




        result = AudienceJobResult.from_dict(d.pop("result"))




        status = AudienceJobStatus(d.pop("status"))




        total = d.pop("total")

        download_url = d.pop("downloadUrl", UNSET)

        error = d.pop("error", UNSET)

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        audience_job = cls(
            created_at=created_at,
            done=done,
            id=id,
            kind=kind,
            result=result,
            status=status,
            total=total,
            download_url=download_url,
            error=error,
            expires_at=expires_at,
            finished_at=finished_at,
        )

        return audience_job

