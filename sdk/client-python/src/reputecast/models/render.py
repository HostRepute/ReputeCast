from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.render_service import RenderService
from ..models.render_status import RenderStatus
from ..models.render_variant import RenderVariant
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.render_result import RenderResult





T = TypeVar("T", bound="Render")



@_attrs_define
class Render:
    """ 
        Attributes:
            id (str):
            requested_at (datetime.datetime):
            requested_by (str):
            results (list[RenderResult]):
            service (RenderService):
            status (RenderStatus):
            variant (RenderVariant):
            version (int): The saved version rendered
            completed_at (datetime.datetime | Unset):
            error (str | Unset):
     """

    id: str
    requested_at: datetime.datetime
    requested_by: str
    results: list[RenderResult]
    service: RenderService
    status: RenderStatus
    variant: RenderVariant
    version: int
    completed_at: datetime.datetime | Unset = UNSET
    error: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.render_result import RenderResult # noqa: PLC0415
        id = self.id

        requested_at = self.requested_at.isoformat()

        requested_by = self.requested_by

        results = []
        for results_item_data in self.results:
            results_item = results_item_data.to_dict()
            results.append(results_item)



        service = self.service.value

        status = self.status.value

        variant = self.variant.value

        version = self.version

        completed_at: str | Unset = UNSET
        if not isinstance(self.completed_at, Unset):
            completed_at = self.completed_at.isoformat()

        error = self.error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "requestedAt": requested_at,
            "requestedBy": requested_by,
            "results": results,
            "service": service,
            "status": status,
            "variant": variant,
            "version": version,
        })
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.render_result import RenderResult # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        requested_by = d.pop("requestedBy")

        results = []
        _results = d.pop("results")
        for results_item_data in (_results):
            results_item = RenderResult.from_dict(results_item_data)



            results.append(results_item)


        service = RenderService(d.pop("service"))




        status = RenderStatus(d.pop("status"))




        variant = RenderVariant(d.pop("variant"))




        version = d.pop("version")

        _completed_at = d.pop("completedAt", UNSET)
        completed_at: datetime.datetime | Unset
        if isinstance(_completed_at,  Unset):
            completed_at = UNSET
        else:
            completed_at = datetime.datetime.fromisoformat(_completed_at)




        error = d.pop("error", UNSET)

        render = cls(
            id=id,
            requested_at=requested_at,
            requested_by=requested_by,
            results=results,
            service=service,
            status=status,
            variant=variant,
            version=version,
            completed_at=completed_at,
            error=error,
        )

        return render

