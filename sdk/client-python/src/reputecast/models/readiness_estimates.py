from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.readiness_audience import ReadinessAudience
  from ..models.readiness_confirm import ReadinessConfirm
  from ..models.readiness_route import ReadinessRoute
  from ..models.readiness_throttle import ReadinessThrottle





T = TypeVar("T", bound="ReadinessEstimates")



@_attrs_define
class ReadinessEstimates:
    """ 
        Attributes:
            audience (ReadinessAudience):
            route (ReadinessRoute):
            throttling (list[ReadinessThrottle]):
            confirm (ReadinessConfirm | Unset):
            hourly_at (datetime.datetime | Unset):
     """

    audience: ReadinessAudience
    route: ReadinessRoute
    throttling: list[ReadinessThrottle]
    confirm: ReadinessConfirm | Unset = UNSET
    hourly_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_audience import ReadinessAudience # noqa: PLC0415
        from ..models.readiness_confirm import ReadinessConfirm # noqa: PLC0415
        from ..models.readiness_route import ReadinessRoute # noqa: PLC0415
        from ..models.readiness_throttle import ReadinessThrottle # noqa: PLC0415
        audience = self.audience.to_dict()

        route = self.route.to_dict()

        throttling = []
        for throttling_item_data in self.throttling:
            throttling_item = throttling_item_data.to_dict()
            throttling.append(throttling_item)



        confirm: dict[str, Any] | Unset = UNSET
        if not isinstance(self.confirm, Unset):
            confirm = self.confirm.to_dict()

        hourly_at: str | Unset = UNSET
        if not isinstance(self.hourly_at, Unset):
            hourly_at = self.hourly_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audience": audience,
            "route": route,
            "throttling": throttling,
        })
        if confirm is not UNSET:
            field_dict["confirm"] = confirm
        if hourly_at is not UNSET:
            field_dict["hourlyAt"] = hourly_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_audience import ReadinessAudience # noqa: PLC0415
        from ..models.readiness_confirm import ReadinessConfirm # noqa: PLC0415
        from ..models.readiness_route import ReadinessRoute # noqa: PLC0415
        from ..models.readiness_throttle import ReadinessThrottle # noqa: PLC0415
        d = dict(src_dict)
        audience = ReadinessAudience.from_dict(d.pop("audience"))




        route = ReadinessRoute.from_dict(d.pop("route"))




        throttling = []
        _throttling = d.pop("throttling")
        for throttling_item_data in (_throttling):
            throttling_item = ReadinessThrottle.from_dict(throttling_item_data)



            throttling.append(throttling_item)


        _confirm = d.pop("confirm", UNSET)
        confirm: ReadinessConfirm | Unset
        if isinstance(_confirm,  Unset):
            confirm = UNSET
        else:
            confirm = ReadinessConfirm.from_dict(_confirm)




        _hourly_at = d.pop("hourlyAt", UNSET)
        hourly_at: datetime.datetime | Unset
        if isinstance(_hourly_at,  Unset):
            hourly_at = UNSET
        else:
            hourly_at = datetime.datetime.fromisoformat(_hourly_at)




        readiness_estimates = cls(
            audience=audience,
            route=route,
            throttling=throttling,
            confirm=confirm,
            hourly_at=hourly_at,
        )

        return readiness_estimates

