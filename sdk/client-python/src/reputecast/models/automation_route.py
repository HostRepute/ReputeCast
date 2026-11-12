from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_route_provider import AutomationRouteProvider





T = TypeVar("T", bound="AutomationRoute")



@_attrs_define
class AutomationRoute:
    """ 
        Attributes:
            healthy (bool): A provider on the route can send right now
            label (str):  Example: SES, failover Postmark.
            primary (list[AutomationRouteProvider]):
            fallback (AutomationRouteProvider | Unset):
     """

    healthy: bool
    label: str
    primary: list[AutomationRouteProvider]
    fallback: AutomationRouteProvider | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_route_provider import AutomationRouteProvider # noqa: PLC0415
        healthy = self.healthy

        label = self.label

        primary = []
        for primary_item_data in self.primary:
            primary_item = primary_item_data.to_dict()
            primary.append(primary_item)



        fallback: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fallback, Unset):
            fallback = self.fallback.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "healthy": healthy,
            "label": label,
            "primary": primary,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_route_provider import AutomationRouteProvider # noqa: PLC0415
        d = dict(src_dict)
        healthy = d.pop("healthy")

        label = d.pop("label")

        primary = []
        _primary = d.pop("primary")
        for primary_item_data in (_primary):
            primary_item = AutomationRouteProvider.from_dict(primary_item_data)



            primary.append(primary_item)


        _fallback = d.pop("fallback", UNSET)
        fallback: AutomationRouteProvider | Unset
        if isinstance(_fallback,  Unset):
            fallback = UNSET
        else:
            fallback = AutomationRouteProvider.from_dict(_fallback)




        automation_route = cls(
            healthy=healthy,
            label=label,
            primary=primary,
            fallback=fallback,
        )

        return automation_route

