from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_routes_input_other import SmsRoutesInputOther
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_route_input import SmsRouteInput





T = TypeVar("T", bound="SmsRoutesInput")



@_attrs_define
class SmsRoutesInput:
    """ 
        Attributes:
            other (SmsRoutesInputOther):
            routes (list[SmsRouteInput]):
            other_route (int | Unset):
     """

    other: SmsRoutesInputOther
    routes: list[SmsRouteInput]
    other_route: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_route_input import SmsRouteInput # noqa: PLC0415
        other = self.other.value

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        other_route = self.other_route


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "other": other,
            "routes": routes,
        })
        if other_route is not UNSET:
            field_dict["otherRoute"] = other_route

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_route_input import SmsRouteInput # noqa: PLC0415
        d = dict(src_dict)
        other = SmsRoutesInputOther(d.pop("other"))




        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = SmsRouteInput.from_dict(routes_item_data)



            routes.append(routes_item)


        other_route = d.pop("otherRoute", UNSET)

        sms_routes_input = cls(
            other=other,
            routes=routes,
            other_route=other_route,
        )

        return sms_routes_input

