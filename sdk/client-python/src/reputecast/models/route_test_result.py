from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_test_result_color import RouteTestResultColor
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.route_test_cost import RouteTestCost
  from ..models.route_test_fallback import RouteTestFallback
  from ..models.route_test_step import RouteTestStep





T = TypeVar("T", bound="RouteTestResult")



@_attrs_define
class RouteTestResult:
    """ 
        Attributes:
            color (RouteTestResultColor):
            cost (RouteTestCost):
            dkim_selector (str):
            fallback (RouteTestFallback):
            held (bool):
            mailbox_provider (str):
            provider (str):
            provider_id (str):
            short (str):
            steps (list[RouteTestStep]):
            suppressed (bool):
            mailbox_detail (str | Unset):
            mx (str | Unset):
     """

    color: RouteTestResultColor
    cost: RouteTestCost
    dkim_selector: str
    fallback: RouteTestFallback
    held: bool
    mailbox_provider: str
    provider: str
    provider_id: str
    short: str
    steps: list[RouteTestStep]
    suppressed: bool
    mailbox_detail: str | Unset = UNSET
    mx: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_test_cost import RouteTestCost # noqa: PLC0415
        from ..models.route_test_fallback import RouteTestFallback # noqa: PLC0415
        from ..models.route_test_step import RouteTestStep # noqa: PLC0415
        color = self.color.value

        cost = self.cost.to_dict()

        dkim_selector = self.dkim_selector

        fallback = self.fallback.to_dict()

        held = self.held

        mailbox_provider = self.mailbox_provider

        provider = self.provider

        provider_id = self.provider_id

        short = self.short

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        suppressed = self.suppressed

        mailbox_detail = self.mailbox_detail

        mx = self.mx


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "cost": cost,
            "dkimSelector": dkim_selector,
            "fallback": fallback,
            "held": held,
            "mailboxProvider": mailbox_provider,
            "provider": provider,
            "providerId": provider_id,
            "short": short,
            "steps": steps,
            "suppressed": suppressed,
        })
        if mailbox_detail is not UNSET:
            field_dict["mailboxDetail"] = mailbox_detail
        if mx is not UNSET:
            field_dict["mx"] = mx

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_test_cost import RouteTestCost # noqa: PLC0415
        from ..models.route_test_fallback import RouteTestFallback # noqa: PLC0415
        from ..models.route_test_step import RouteTestStep # noqa: PLC0415
        d = dict(src_dict)
        color = RouteTestResultColor(d.pop("color"))




        cost = RouteTestCost.from_dict(d.pop("cost"))




        dkim_selector = d.pop("dkimSelector")

        fallback = RouteTestFallback.from_dict(d.pop("fallback"))




        held = d.pop("held")

        mailbox_provider = d.pop("mailboxProvider")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = RouteTestStep.from_dict(steps_item_data)



            steps.append(steps_item)


        suppressed = d.pop("suppressed")

        mailbox_detail = d.pop("mailboxDetail", UNSET)

        mx = d.pop("mx", UNSET)

        route_test_result = cls(
            color=color,
            cost=cost,
            dkim_selector=dkim_selector,
            fallback=fallback,
            held=held,
            mailbox_provider=mailbox_provider,
            provider=provider,
            provider_id=provider_id,
            short=short,
            steps=steps,
            suppressed=suppressed,
            mailbox_detail=mailbox_detail,
            mx=mx,
        )

        return route_test_result

