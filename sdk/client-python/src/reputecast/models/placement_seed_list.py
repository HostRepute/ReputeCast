from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_seed_list_delete_after import PlacementSeedListDeleteAfter
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_seed_provider import PlacementSeedProvider
  from ..models.placement_seed_view import PlacementSeedView





T = TypeVar("T", bound="PlacementSeedList")



@_attrs_define
class PlacementSeedList:
    """ 
        Attributes:
            attention (int): Seeds whose sign-in failed or expired
            can_manage (bool):
            can_run (bool):
            connected (int):
            delete_after (PlacementSeedListDeleteAfter):
            manage_helpers (list[str]):
            oauth_google (bool):
            oauth_microsoft (bool):
            providers (list[PlacementSeedProvider]):
            redirect_uri (str):
            run_helpers (list[str]):
            seeds (list[PlacementSeedView]):
     """

    attention: int
    can_manage: bool
    can_run: bool
    connected: int
    delete_after: PlacementSeedListDeleteAfter
    manage_helpers: list[str]
    oauth_google: bool
    oauth_microsoft: bool
    providers: list[PlacementSeedProvider]
    redirect_uri: str
    run_helpers: list[str]
    seeds: list[PlacementSeedView]





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_seed_provider import PlacementSeedProvider # noqa: PLC0415
        from ..models.placement_seed_view import PlacementSeedView # noqa: PLC0415
        attention = self.attention

        can_manage = self.can_manage

        can_run = self.can_run

        connected = self.connected

        delete_after = self.delete_after.value

        manage_helpers = self.manage_helpers



        oauth_google = self.oauth_google

        oauth_microsoft = self.oauth_microsoft

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        redirect_uri = self.redirect_uri

        run_helpers = self.run_helpers



        seeds = []
        for seeds_item_data in self.seeds:
            seeds_item = seeds_item_data.to_dict()
            seeds.append(seeds_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attention": attention,
            "canManage": can_manage,
            "canRun": can_run,
            "connected": connected,
            "deleteAfter": delete_after,
            "manageHelpers": manage_helpers,
            "oauthGoogle": oauth_google,
            "oauthMicrosoft": oauth_microsoft,
            "providers": providers,
            "redirectUri": redirect_uri,
            "runHelpers": run_helpers,
            "seeds": seeds,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_seed_provider import PlacementSeedProvider # noqa: PLC0415
        from ..models.placement_seed_view import PlacementSeedView # noqa: PLC0415
        d = dict(src_dict)
        attention = d.pop("attention")

        can_manage = d.pop("canManage")

        can_run = d.pop("canRun")

        connected = d.pop("connected")

        delete_after = PlacementSeedListDeleteAfter(d.pop("deleteAfter"))




        manage_helpers = cast(list[str], d.pop("manageHelpers"))


        oauth_google = d.pop("oauthGoogle")

        oauth_microsoft = d.pop("oauthMicrosoft")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = PlacementSeedProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        redirect_uri = d.pop("redirectUri")

        run_helpers = cast(list[str], d.pop("runHelpers"))


        seeds = []
        _seeds = d.pop("seeds")
        for seeds_item_data in (_seeds):
            seeds_item = PlacementSeedView.from_dict(seeds_item_data)



            seeds.append(seeds_item)


        placement_seed_list = cls(
            attention=attention,
            can_manage=can_manage,
            can_run=can_run,
            connected=connected,
            delete_after=delete_after,
            manage_helpers=manage_helpers,
            oauth_google=oauth_google,
            oauth_microsoft=oauth_microsoft,
            providers=providers,
            redirect_uri=redirect_uri,
            run_helpers=run_helpers,
            seeds=seeds,
        )

        return placement_seed_list

