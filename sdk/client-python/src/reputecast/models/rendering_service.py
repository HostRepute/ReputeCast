from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.rendering_service_region import RenderingServiceRegion
from ..models.rendering_service_service import RenderingServiceService
from ..models.rendering_service_status import RenderingServiceStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.render_client import RenderClient





T = TypeVar("T", bound="RenderingService")



@_attrs_define
class RenderingService:
    """ 
        Attributes:
            clients (list[RenderClient]): The email clients the account can render, as the service last listed them
            connected (bool):
            checked_at (datetime.datetime | Unset):
            connected_at (datetime.datetime | Unset):
            connected_by (str | Unset):
            key_hint (str | Unset):
            last_error (str | Unset):
            region (RenderingServiceRegion | Unset): Mailgun Inspect: the account's region
            service (RenderingServiceService | Unset):
            status (RenderingServiceStatus | Unset): failing: the last call was refused or failed (lastError says why)
     """

    clients: list[RenderClient]
    connected: bool
    checked_at: datetime.datetime | Unset = UNSET
    connected_at: datetime.datetime | Unset = UNSET
    connected_by: str | Unset = UNSET
    key_hint: str | Unset = UNSET
    last_error: str | Unset = UNSET
    region: RenderingServiceRegion | Unset = UNSET
    service: RenderingServiceService | Unset = UNSET
    status: RenderingServiceStatus | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.render_client import RenderClient # noqa: PLC0415
        clients = []
        for clients_item_data in self.clients:
            clients_item = clients_item_data.to_dict()
            clients.append(clients_item)



        connected = self.connected

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        key_hint = self.key_hint

        last_error = self.last_error

        region: str | Unset = UNSET
        if not isinstance(self.region, Unset):
            region = self.region.value


        service: str | Unset = UNSET
        if not isinstance(self.service, Unset):
            service = self.service.value


        status: str | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clients": clients,
            "connected": connected,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if key_hint is not UNSET:
            field_dict["keyHint"] = key_hint
        if last_error is not UNSET:
            field_dict["lastError"] = last_error
        if region is not UNSET:
            field_dict["region"] = region
        if service is not UNSET:
            field_dict["service"] = service
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.render_client import RenderClient # noqa: PLC0415
        d = dict(src_dict)
        clients = []
        _clients = d.pop("clients")
        for clients_item_data in (_clients):
            clients_item = RenderClient.from_dict(clients_item_data)



            clients.append(clients_item)


        connected = d.pop("connected")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        connected_by = d.pop("connectedBy", UNSET)

        key_hint = d.pop("keyHint", UNSET)

        last_error = d.pop("lastError", UNSET)

        _region = d.pop("region", UNSET)
        region: RenderingServiceRegion | Unset
        if isinstance(_region,  Unset):
            region = UNSET
        else:
            region = RenderingServiceRegion(_region)




        _service = d.pop("service", UNSET)
        service: RenderingServiceService | Unset
        if isinstance(_service,  Unset):
            service = UNSET
        else:
            service = RenderingServiceService(_service)




        _status = d.pop("status", UNSET)
        status: RenderingServiceStatus | Unset
        if isinstance(_status,  Unset):
            status = UNSET
        else:
            status = RenderingServiceStatus(_status)




        rendering_service = cls(
            clients=clients,
            connected=connected,
            checked_at=checked_at,
            connected_at=connected_at,
            connected_by=connected_by,
            key_hint=key_hint,
            last_error=last_error,
            region=region,
            service=service,
            status=status,
        )

        return rendering_service

