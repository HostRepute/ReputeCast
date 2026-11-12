from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_call_view_who import IntegrationCallViewWho
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.call_target import CallTarget
  from ..models.integration_resolution import IntegrationResolution





T = TypeVar("T", bound="IntegrationCallView")



@_attrs_define
class IntegrationCallView:
    """ 
        Attributes:
            at (datetime.datetime):
            duration_ms (int):
            id (str):
            method (str):
            operation (str):
            path (str):
            status (int):
            target (CallTarget):
            unnamed (bool):
            who (IntegrationCallViewWho):
            workflow (str):
            client (str | Unset):
            code (str | Unset):
            detail (str | Unset):
            ip (str | Unset):
            item (str | Unset):
            missing (list[str] | Unset):
            resolution (IntegrationResolution | Unset):
     """

    at: datetime.datetime
    duration_ms: int
    id: str
    method: str
    operation: str
    path: str
    status: int
    target: CallTarget
    unnamed: bool
    who: IntegrationCallViewWho
    workflow: str
    client: str | Unset = UNSET
    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    ip: str | Unset = UNSET
    item: str | Unset = UNSET
    missing: list[str] | Unset = UNSET
    resolution: IntegrationResolution | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.call_target import CallTarget # noqa: PLC0415
        from ..models.integration_resolution import IntegrationResolution # noqa: PLC0415
        at = self.at.isoformat()

        duration_ms = self.duration_ms

        id = self.id

        method = self.method

        operation = self.operation

        path = self.path

        status = self.status

        target = self.target.to_dict()

        unnamed = self.unnamed

        who = self.who.value

        workflow = self.workflow

        client = self.client

        code = self.code

        detail = self.detail

        ip = self.ip

        item = self.item

        missing: list[str] | Unset = UNSET
        if not isinstance(self.missing, Unset):
            missing = self.missing



        resolution: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resolution, Unset):
            resolution = self.resolution.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "durationMs": duration_ms,
            "id": id,
            "method": method,
            "operation": operation,
            "path": path,
            "status": status,
            "target": target,
            "unnamed": unnamed,
            "who": who,
            "workflow": workflow,
        })
        if client is not UNSET:
            field_dict["client"] = client
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail
        if ip is not UNSET:
            field_dict["ip"] = ip
        if item is not UNSET:
            field_dict["item"] = item
        if missing is not UNSET:
            field_dict["missing"] = missing
        if resolution is not UNSET:
            field_dict["resolution"] = resolution

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.call_target import CallTarget # noqa: PLC0415
        from ..models.integration_resolution import IntegrationResolution # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        duration_ms = d.pop("durationMs")

        id = d.pop("id")

        method = d.pop("method")

        operation = d.pop("operation")

        path = d.pop("path")

        status = d.pop("status")

        target = CallTarget.from_dict(d.pop("target"))




        unnamed = d.pop("unnamed")

        who = IntegrationCallViewWho(d.pop("who"))




        workflow = d.pop("workflow")

        client = d.pop("client", UNSET)

        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        ip = d.pop("ip", UNSET)

        item = d.pop("item", UNSET)

        missing = cast(list[str], d.pop("missing", UNSET))


        _resolution = d.pop("resolution", UNSET)
        resolution: IntegrationResolution | Unset
        if isinstance(_resolution,  Unset):
            resolution = UNSET
        else:
            resolution = IntegrationResolution.from_dict(_resolution)




        integration_call_view = cls(
            at=at,
            duration_ms=duration_ms,
            id=id,
            method=method,
            operation=operation,
            path=path,
            status=status,
            target=target,
            unnamed=unnamed,
            who=who,
            workflow=workflow,
            client=client,
            code=code,
            detail=detail,
            ip=ip,
            item=item,
            missing=missing,
            resolution=resolution,
        )

        return integration_call_view

