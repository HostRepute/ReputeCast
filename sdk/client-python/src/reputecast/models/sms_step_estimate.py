from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.country_line import CountryLine
  from ..models.sms_budget_view import SmsBudgetView
  from ..models.sms_counter import SmsCounter
  from ..models.sms_link_view import SmsLinkView





T = TypeVar("T", bound="SmsStepEstimate")



@_attrs_define
class SmsStepEstimate:
    """ 
        Attributes:
            budget (SmsBudgetView):
            cost (float):
            costs (list[CountryLine]):
            counter (SmsCounter):
            links (list[SmsLinkView]):
            opt_out (str):
            prefix (str):
            short_domain (str):
            shorteners (list[str]):
            texts (int):
            window (str):
     """

    budget: SmsBudgetView
    cost: float
    costs: list[CountryLine]
    counter: SmsCounter
    links: list[SmsLinkView]
    opt_out: str
    prefix: str
    short_domain: str
    shorteners: list[str]
    texts: int
    window: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.country_line import CountryLine # noqa: PLC0415
        from ..models.sms_budget_view import SmsBudgetView # noqa: PLC0415
        from ..models.sms_counter import SmsCounter # noqa: PLC0415
        from ..models.sms_link_view import SmsLinkView # noqa: PLC0415
        budget = self.budget.to_dict()

        cost = self.cost

        costs = []
        for costs_item_data in self.costs:
            costs_item = costs_item_data.to_dict()
            costs.append(costs_item)



        counter = self.counter.to_dict()

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        opt_out = self.opt_out

        prefix = self.prefix

        short_domain = self.short_domain

        shorteners = self.shorteners



        texts = self.texts

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "budget": budget,
            "cost": cost,
            "costs": costs,
            "counter": counter,
            "links": links,
            "optOut": opt_out,
            "prefix": prefix,
            "shortDomain": short_domain,
            "shorteners": shorteners,
            "texts": texts,
            "window": window,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.country_line import CountryLine # noqa: PLC0415
        from ..models.sms_budget_view import SmsBudgetView # noqa: PLC0415
        from ..models.sms_counter import SmsCounter # noqa: PLC0415
        from ..models.sms_link_view import SmsLinkView # noqa: PLC0415
        d = dict(src_dict)
        budget = SmsBudgetView.from_dict(d.pop("budget"))




        cost = d.pop("cost")

        costs = []
        _costs = d.pop("costs")
        for costs_item_data in (_costs):
            costs_item = CountryLine.from_dict(costs_item_data)



            costs.append(costs_item)


        counter = SmsCounter.from_dict(d.pop("counter"))




        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = SmsLinkView.from_dict(links_item_data)



            links.append(links_item)


        opt_out = d.pop("optOut")

        prefix = d.pop("prefix")

        short_domain = d.pop("shortDomain")

        shorteners = cast(list[str], d.pop("shorteners"))


        texts = d.pop("texts")

        window = d.pop("window")

        sms_step_estimate = cls(
            budget=budget,
            cost=cost,
            costs=costs,
            counter=counter,
            links=links,
            opt_out=opt_out,
            prefix=prefix,
            short_domain=short_domain,
            shorteners=shorteners,
            texts=texts,
            window=window,
        )

        return sms_step_estimate

