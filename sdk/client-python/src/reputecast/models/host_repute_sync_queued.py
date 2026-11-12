from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="HostReputeSyncQueued")



@_attrs_define
class HostReputeSyncQueued:
    """ 
        Attributes:
            run_id (str):
            started_at (datetime.datetime):
     """

    run_id: str
    started_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        run_id = self.run_id

        started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "runId": run_id,
            "startedAt": started_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        run_id = d.pop("runId")

        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        host_repute_sync_queued = cls(
            run_id=run_id,
            started_at=started_at,
        )

        return host_repute_sync_queued

