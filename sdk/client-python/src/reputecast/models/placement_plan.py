from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_plan_stream import PlacementPlanStream
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_content_view import PlacementContentView
  from ..models.placement_copy_cost import PlacementCopyCost
  from ..models.placement_provider_option import PlacementProviderOption
  from ..models.placement_route_share import PlacementRouteShare
  from ..models.placement_seed_choice import PlacementSeedChoice
  from ..models.placement_sender_option import PlacementSenderOption





T = TypeVar("T", bound="PlacementPlan")



@_attrs_define
class PlacementPlan:
    """ 
        Attributes:
            can_run (bool):
            can_start (bool):
            chosen (list[str]):
            content (PlacementContentView):
            copies (int):
            costs (list[PlacementCopyCost]):
            problems (list[str]):
            providers (list[PlacementProviderOption]):
            route (list[PlacementRouteShare]):
            run_helpers (list[str]):
            seeds (list[PlacementSeedChoice]):
            senders (list[PlacementSenderOption]):
            stream (PlacementPlanStream):
            warmup_days (list[int]):
            alone (str | Unset):
            compare_problem (str | Unset):
            sender (str | Unset):
            sender_alternatives (list[str] | Unset):
            sender_domain (str | Unset):
            sender_missing (list[str] | Unset):
            sender_problem (str | Unset):
     """

    can_run: bool
    can_start: bool
    chosen: list[str]
    content: PlacementContentView
    copies: int
    costs: list[PlacementCopyCost]
    problems: list[str]
    providers: list[PlacementProviderOption]
    route: list[PlacementRouteShare]
    run_helpers: list[str]
    seeds: list[PlacementSeedChoice]
    senders: list[PlacementSenderOption]
    stream: PlacementPlanStream
    warmup_days: list[int]
    alone: str | Unset = UNSET
    compare_problem: str | Unset = UNSET
    sender: str | Unset = UNSET
    sender_alternatives: list[str] | Unset = UNSET
    sender_domain: str | Unset = UNSET
    sender_missing: list[str] | Unset = UNSET
    sender_problem: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_content_view import PlacementContentView # noqa: PLC0415
        from ..models.placement_copy_cost import PlacementCopyCost # noqa: PLC0415
        from ..models.placement_provider_option import PlacementProviderOption # noqa: PLC0415
        from ..models.placement_route_share import PlacementRouteShare # noqa: PLC0415
        from ..models.placement_seed_choice import PlacementSeedChoice # noqa: PLC0415
        from ..models.placement_sender_option import PlacementSenderOption # noqa: PLC0415
        can_run = self.can_run

        can_start = self.can_start

        chosen = self.chosen



        content = self.content.to_dict()

        copies = self.copies

        costs = []
        for costs_item_data in self.costs:
            costs_item = costs_item_data.to_dict()
            costs.append(costs_item)



        problems = self.problems



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        route = []
        for route_item_data in self.route:
            route_item = route_item_data.to_dict()
            route.append(route_item)



        run_helpers = self.run_helpers



        seeds = []
        for seeds_item_data in self.seeds:
            seeds_item = seeds_item_data.to_dict()
            seeds.append(seeds_item)



        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        stream = self.stream.value

        warmup_days = self.warmup_days



        alone = self.alone

        compare_problem = self.compare_problem

        sender = self.sender

        sender_alternatives: list[str] | Unset = UNSET
        if not isinstance(self.sender_alternatives, Unset):
            sender_alternatives = self.sender_alternatives



        sender_domain = self.sender_domain

        sender_missing: list[str] | Unset = UNSET
        if not isinstance(self.sender_missing, Unset):
            sender_missing = self.sender_missing



        sender_problem = self.sender_problem


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canRun": can_run,
            "canStart": can_start,
            "chosen": chosen,
            "content": content,
            "copies": copies,
            "costs": costs,
            "problems": problems,
            "providers": providers,
            "route": route,
            "runHelpers": run_helpers,
            "seeds": seeds,
            "senders": senders,
            "stream": stream,
            "warmupDays": warmup_days,
        })
        if alone is not UNSET:
            field_dict["alone"] = alone
        if compare_problem is not UNSET:
            field_dict["compareProblem"] = compare_problem
        if sender is not UNSET:
            field_dict["sender"] = sender
        if sender_alternatives is not UNSET:
            field_dict["senderAlternatives"] = sender_alternatives
        if sender_domain is not UNSET:
            field_dict["senderDomain"] = sender_domain
        if sender_missing is not UNSET:
            field_dict["senderMissing"] = sender_missing
        if sender_problem is not UNSET:
            field_dict["senderProblem"] = sender_problem

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_content_view import PlacementContentView # noqa: PLC0415
        from ..models.placement_copy_cost import PlacementCopyCost # noqa: PLC0415
        from ..models.placement_provider_option import PlacementProviderOption # noqa: PLC0415
        from ..models.placement_route_share import PlacementRouteShare # noqa: PLC0415
        from ..models.placement_seed_choice import PlacementSeedChoice # noqa: PLC0415
        from ..models.placement_sender_option import PlacementSenderOption # noqa: PLC0415
        d = dict(src_dict)
        can_run = d.pop("canRun")

        can_start = d.pop("canStart")

        chosen = cast(list[str], d.pop("chosen"))


        content = PlacementContentView.from_dict(d.pop("content"))




        copies = d.pop("copies")

        costs = []
        _costs = d.pop("costs")
        for costs_item_data in (_costs):
            costs_item = PlacementCopyCost.from_dict(costs_item_data)



            costs.append(costs_item)


        problems = cast(list[str], d.pop("problems"))


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = PlacementProviderOption.from_dict(providers_item_data)



            providers.append(providers_item)


        route = []
        _route = d.pop("route")
        for route_item_data in (_route):
            route_item = PlacementRouteShare.from_dict(route_item_data)



            route.append(route_item)


        run_helpers = cast(list[str], d.pop("runHelpers"))


        seeds = []
        _seeds = d.pop("seeds")
        for seeds_item_data in (_seeds):
            seeds_item = PlacementSeedChoice.from_dict(seeds_item_data)



            seeds.append(seeds_item)


        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = PlacementSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        stream = PlacementPlanStream(d.pop("stream"))




        warmup_days = cast(list[int], d.pop("warmupDays"))


        alone = d.pop("alone", UNSET)

        compare_problem = d.pop("compareProblem", UNSET)

        sender = d.pop("sender", UNSET)

        sender_alternatives = cast(list[str], d.pop("senderAlternatives", UNSET))


        sender_domain = d.pop("senderDomain", UNSET)

        sender_missing = cast(list[str], d.pop("senderMissing", UNSET))


        sender_problem = d.pop("senderProblem", UNSET)

        placement_plan = cls(
            can_run=can_run,
            can_start=can_start,
            chosen=chosen,
            content=content,
            copies=copies,
            costs=costs,
            problems=problems,
            providers=providers,
            route=route,
            run_helpers=run_helpers,
            seeds=seeds,
            senders=senders,
            stream=stream,
            warmup_days=warmup_days,
            alone=alone,
            compare_problem=compare_problem,
            sender=sender,
            sender_alternatives=sender_alternatives,
            sender_domain=sender_domain,
            sender_missing=sender_missing,
            sender_problem=sender_problem,
        )

        return placement_plan

