from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_setup_status import ConnectSetupStatus
from ..models.connect_setup_step import ConnectSetupStep
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.connect_adapter_info import ConnectAdapterInfo
  from ..models.connect_connection import ConnectConnection
  from ..models.connect_domain_choice import ConnectDomainChoice
  from ..models.connect_done import ConnectDone
  from ..models.connect_streams import ConnectStreams
  from ..models.connect_summaries import ConnectSummaries





T = TypeVar("T", bound="ConnectSetup")



@_attrs_define
class ConnectSetup:
    """ 
        Attributes:
            adapter (str):
            can_manage (bool):
            connection (ConnectConnection):
            created_at (datetime.datetime):
            domain (ConnectDomainChoice):
            helpers (list[str]):
            id (str):
            info (ConnectAdapterInfo):
            provider (str):
            provider_id (str):
            status (ConnectSetupStatus):
            step (ConnectSetupStep):
            streams (ConnectStreams):
            summaries (ConnectSummaries):
            updated_at (datetime.datetime):
            done (ConnectDone | Unset):
            finished_at (datetime.datetime | Unset):
     """

    adapter: str
    can_manage: bool
    connection: ConnectConnection
    created_at: datetime.datetime
    domain: ConnectDomainChoice
    helpers: list[str]
    id: str
    info: ConnectAdapterInfo
    provider: str
    provider_id: str
    status: ConnectSetupStatus
    step: ConnectSetupStep
    streams: ConnectStreams
    summaries: ConnectSummaries
    updated_at: datetime.datetime
    done: ConnectDone | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_adapter_info import ConnectAdapterInfo # noqa: PLC0415
        from ..models.connect_connection import ConnectConnection # noqa: PLC0415
        from ..models.connect_domain_choice import ConnectDomainChoice # noqa: PLC0415
        from ..models.connect_done import ConnectDone # noqa: PLC0415
        from ..models.connect_streams import ConnectStreams # noqa: PLC0415
        from ..models.connect_summaries import ConnectSummaries # noqa: PLC0415
        adapter = self.adapter

        can_manage = self.can_manage

        connection = self.connection.to_dict()

        created_at = self.created_at.isoformat()

        domain = self.domain.to_dict()

        helpers = self.helpers



        id = self.id

        info = self.info.to_dict()

        provider = self.provider

        provider_id = self.provider_id

        status = self.status.value

        step = self.step.value

        streams = self.streams.to_dict()

        summaries = self.summaries.to_dict()

        updated_at = self.updated_at.isoformat()

        done: dict[str, Any] | Unset = UNSET
        if not isinstance(self.done, Unset):
            done = self.done.to_dict()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "canManage": can_manage,
            "connection": connection,
            "createdAt": created_at,
            "domain": domain,
            "helpers": helpers,
            "id": id,
            "info": info,
            "provider": provider,
            "providerId": provider_id,
            "status": status,
            "step": step,
            "streams": streams,
            "summaries": summaries,
            "updatedAt": updated_at,
        })
        if done is not UNSET:
            field_dict["done"] = done
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_adapter_info import ConnectAdapterInfo # noqa: PLC0415
        from ..models.connect_connection import ConnectConnection # noqa: PLC0415
        from ..models.connect_domain_choice import ConnectDomainChoice # noqa: PLC0415
        from ..models.connect_done import ConnectDone # noqa: PLC0415
        from ..models.connect_streams import ConnectStreams # noqa: PLC0415
        from ..models.connect_summaries import ConnectSummaries # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        can_manage = d.pop("canManage")

        connection = ConnectConnection.from_dict(d.pop("connection"))




        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        domain = ConnectDomainChoice.from_dict(d.pop("domain"))




        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        info = ConnectAdapterInfo.from_dict(d.pop("info"))




        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        status = ConnectSetupStatus(d.pop("status"))




        step = ConnectSetupStep(d.pop("step"))




        streams = ConnectStreams.from_dict(d.pop("streams"))




        summaries = ConnectSummaries.from_dict(d.pop("summaries"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _done = d.pop("done", UNSET)
        done: ConnectDone | Unset
        if isinstance(_done,  Unset):
            done = UNSET
        else:
            done = ConnectDone.from_dict(_done)




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        connect_setup = cls(
            adapter=adapter,
            can_manage=can_manage,
            connection=connection,
            created_at=created_at,
            domain=domain,
            helpers=helpers,
            id=id,
            info=info,
            provider=provider,
            provider_id=provider_id,
            status=status,
            step=step,
            streams=streams,
            summaries=summaries,
            updated_at=updated_at,
            done=done,
            finished_at=finished_at,
        )

        return connect_setup

