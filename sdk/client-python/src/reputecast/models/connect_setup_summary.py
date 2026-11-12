from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_setup_summary_status import ConnectSetupSummaryStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectSetupSummary")



@_attrs_define
class ConnectSetupSummary:
    """ 
        Attributes:
            adapter (str):
            created_at (datetime.datetime):
            id (str):
            name (str):
            provider (str):
            status (ConnectSetupSummaryStatus):
            step (str):
            updated_at (datetime.datetime):
            finished_at (datetime.datetime | Unset):
     """

    adapter: str
    created_at: datetime.datetime
    id: str
    name: str
    provider: str
    status: ConnectSetupSummaryStatus
    step: str
    updated_at: datetime.datetime
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        provider = self.provider

        status = self.status.value

        step = self.step

        updated_at = self.updated_at.isoformat()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "createdAt": created_at,
            "id": id,
            "name": name,
            "provider": provider,
            "status": status,
            "step": step,
            "updatedAt": updated_at,
        })
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        id = d.pop("id")

        name = d.pop("name")

        provider = d.pop("provider")

        status = ConnectSetupSummaryStatus(d.pop("status"))




        step = d.pop("step")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        connect_setup_summary = cls(
            adapter=adapter,
            created_at=created_at,
            id=id,
            name=name,
            provider=provider,
            status=status,
            step=step,
            updated_at=updated_at,
            finished_at=finished_at,
        )

        return connect_setup_summary

