from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_worker_mode import PushWorkerMode
from ..models.push_worker_status import PushWorkerStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushWorker")



@_attrs_define
class PushWorker:
    """ 
        Attributes:
            current (int):
            mode (PushWorkerMode):
            path (str):
            scope (str):
            status (PushWorkerStatus):
            url (str):
            checked_at (datetime.datetime | Unset):
            detail (str | Unset):
            http_status (int | Unset):
            version (int | Unset):
     """

    current: int
    mode: PushWorkerMode
    path: str
    scope: str
    status: PushWorkerStatus
    url: str
    checked_at: datetime.datetime | Unset = UNSET
    detail: str | Unset = UNSET
    http_status: int | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        current = self.current

        mode = self.mode.value

        path = self.path

        scope = self.scope

        status = self.status.value

        url = self.url

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        detail = self.detail

        http_status = self.http_status

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "current": current,
            "mode": mode,
            "path": path,
            "scope": scope,
            "status": status,
            "url": url,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if detail is not UNSET:
            field_dict["detail"] = detail
        if http_status is not UNSET:
            field_dict["httpStatus"] = http_status
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        current = d.pop("current")

        mode = PushWorkerMode(d.pop("mode"))




        path = d.pop("path")

        scope = d.pop("scope")

        status = PushWorkerStatus(d.pop("status"))




        url = d.pop("url")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        detail = d.pop("detail", UNSET)

        http_status = d.pop("httpStatus", UNSET)

        version = d.pop("version", UNSET)

        push_worker = cls(
            current=current,
            mode=mode,
            path=path,
            scope=scope,
            status=status,
            url=url,
            checked_at=checked_at,
            detail=detail,
            http_status=http_status,
            version=version,
        )

        return push_worker

