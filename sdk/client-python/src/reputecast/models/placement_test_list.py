from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_test_list_delete_after import PlacementTestListDeleteAfter
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_facet import PlacementFacet
  from ..models.placement_seed_view import PlacementSeedView
  from ..models.placement_test_row import PlacementTestRow





T = TypeVar("T", bound="PlacementTestList")



@_attrs_define
class PlacementTestList:
    """ 
        Attributes:
            attention (list[PlacementSeedView]):
            can_manage (bool):
            can_run (bool):
            delete_after (PlacementTestListDeleteAfter):
            manage_helpers (list[str]):
            providers (list[PlacementFacet]):
            run_helpers (list[str]):
            seeds_connected (int):
            seeds_total (int):
            starters (list[PlacementFacet]):
            tests (list[PlacementTestRow]):
     """

    attention: list[PlacementSeedView]
    can_manage: bool
    can_run: bool
    delete_after: PlacementTestListDeleteAfter
    manage_helpers: list[str]
    providers: list[PlacementFacet]
    run_helpers: list[str]
    seeds_connected: int
    seeds_total: int
    starters: list[PlacementFacet]
    tests: list[PlacementTestRow]





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_facet import PlacementFacet # noqa: PLC0415
        from ..models.placement_seed_view import PlacementSeedView # noqa: PLC0415
        from ..models.placement_test_row import PlacementTestRow # noqa: PLC0415
        attention = []
        for attention_item_data in self.attention:
            attention_item = attention_item_data.to_dict()
            attention.append(attention_item)



        can_manage = self.can_manage

        can_run = self.can_run

        delete_after = self.delete_after.value

        manage_helpers = self.manage_helpers



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        run_helpers = self.run_helpers



        seeds_connected = self.seeds_connected

        seeds_total = self.seeds_total

        starters = []
        for starters_item_data in self.starters:
            starters_item = starters_item_data.to_dict()
            starters.append(starters_item)



        tests = []
        for tests_item_data in self.tests:
            tests_item = tests_item_data.to_dict()
            tests.append(tests_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attention": attention,
            "canManage": can_manage,
            "canRun": can_run,
            "deleteAfter": delete_after,
            "manageHelpers": manage_helpers,
            "providers": providers,
            "runHelpers": run_helpers,
            "seedsConnected": seeds_connected,
            "seedsTotal": seeds_total,
            "starters": starters,
            "tests": tests,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_facet import PlacementFacet # noqa: PLC0415
        from ..models.placement_seed_view import PlacementSeedView # noqa: PLC0415
        from ..models.placement_test_row import PlacementTestRow # noqa: PLC0415
        d = dict(src_dict)
        attention = []
        _attention = d.pop("attention")
        for attention_item_data in (_attention):
            attention_item = PlacementSeedView.from_dict(attention_item_data)



            attention.append(attention_item)


        can_manage = d.pop("canManage")

        can_run = d.pop("canRun")

        delete_after = PlacementTestListDeleteAfter(d.pop("deleteAfter"))




        manage_helpers = cast(list[str], d.pop("manageHelpers"))


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = PlacementFacet.from_dict(providers_item_data)



            providers.append(providers_item)


        run_helpers = cast(list[str], d.pop("runHelpers"))


        seeds_connected = d.pop("seedsConnected")

        seeds_total = d.pop("seedsTotal")

        starters = []
        _starters = d.pop("starters")
        for starters_item_data in (_starters):
            starters_item = PlacementFacet.from_dict(starters_item_data)



            starters.append(starters_item)


        tests = []
        _tests = d.pop("tests")
        for tests_item_data in (_tests):
            tests_item = PlacementTestRow.from_dict(tests_item_data)



            tests.append(tests_item)


        placement_test_list = cls(
            attention=attention,
            can_manage=can_manage,
            can_run=can_run,
            delete_after=delete_after,
            manage_helpers=manage_helpers,
            providers=providers,
            run_helpers=run_helpers,
            seeds_connected=seeds_connected,
            seeds_total=seeds_total,
            starters=starters,
            tests=tests,
        )

        return placement_test_list

