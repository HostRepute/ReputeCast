from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.partner_code_item import PartnerCodeItem
  from ..models.rec_block_use import RecBlockUse
  from ..models.rec_list_option import RecListOption
  from ..models.rec_profile import RecProfile
  from ..models.recommendation_item import RecommendationItem
  from ..models.recommending_you_item import RecommendingYouItem





T = TypeVar("T", bound="RecommendationsView")



@_attrs_define
class RecommendationsView:
    """ 
        Attributes:
            block_shows (int):
            blocks (list[RecBlockUse]):
            codes (list[PartnerCodeItem]):
            confirm_page (bool):
            confirm_shows (list[str]):
            helpers (GrowHelperSets):
            items (list[RecommendationItem]):
            lists (list[RecListOption]):
            page_url (str):
            profile (RecProfile):
            recommending_you (list[RecommendingYouItem]):
     """

    block_shows: int
    blocks: list[RecBlockUse]
    codes: list[PartnerCodeItem]
    confirm_page: bool
    confirm_shows: list[str]
    helpers: GrowHelperSets
    items: list[RecommendationItem]
    lists: list[RecListOption]
    page_url: str
    profile: RecProfile
    recommending_you: list[RecommendingYouItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.partner_code_item import PartnerCodeItem # noqa: PLC0415
        from ..models.rec_block_use import RecBlockUse # noqa: PLC0415
        from ..models.rec_list_option import RecListOption # noqa: PLC0415
        from ..models.rec_profile import RecProfile # noqa: PLC0415
        from ..models.recommendation_item import RecommendationItem # noqa: PLC0415
        from ..models.recommending_you_item import RecommendingYouItem # noqa: PLC0415
        block_shows = self.block_shows

        blocks = []
        for blocks_item_data in self.blocks:
            blocks_item = blocks_item_data.to_dict()
            blocks.append(blocks_item)



        codes = []
        for codes_item_data in self.codes:
            codes_item = codes_item_data.to_dict()
            codes.append(codes_item)



        confirm_page = self.confirm_page

        confirm_shows = self.confirm_shows



        helpers = self.helpers.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        page_url = self.page_url

        profile = self.profile.to_dict()

        recommending_you = []
        for recommending_you_item_data in self.recommending_you:
            recommending_you_item = recommending_you_item_data.to_dict()
            recommending_you.append(recommending_you_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blockShows": block_shows,
            "blocks": blocks,
            "codes": codes,
            "confirmPage": confirm_page,
            "confirmShows": confirm_shows,
            "helpers": helpers,
            "items": items,
            "lists": lists,
            "pageUrl": page_url,
            "profile": profile,
            "recommendingYou": recommending_you,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.partner_code_item import PartnerCodeItem # noqa: PLC0415
        from ..models.rec_block_use import RecBlockUse # noqa: PLC0415
        from ..models.rec_list_option import RecListOption # noqa: PLC0415
        from ..models.rec_profile import RecProfile # noqa: PLC0415
        from ..models.recommendation_item import RecommendationItem # noqa: PLC0415
        from ..models.recommending_you_item import RecommendingYouItem # noqa: PLC0415
        d = dict(src_dict)
        block_shows = d.pop("blockShows")

        blocks = []
        _blocks = d.pop("blocks")
        for blocks_item_data in (_blocks):
            blocks_item = RecBlockUse.from_dict(blocks_item_data)



            blocks.append(blocks_item)


        codes = []
        _codes = d.pop("codes")
        for codes_item_data in (_codes):
            codes_item = PartnerCodeItem.from_dict(codes_item_data)



            codes.append(codes_item)


        confirm_page = d.pop("confirmPage")

        confirm_shows = cast(list[str], d.pop("confirmShows"))


        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = RecommendationItem.from_dict(items_item_data)



            items.append(items_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = RecListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        page_url = d.pop("pageUrl")

        profile = RecProfile.from_dict(d.pop("profile"))




        recommending_you = []
        _recommending_you = d.pop("recommendingYou")
        for recommending_you_item_data in (_recommending_you):
            recommending_you_item = RecommendingYouItem.from_dict(recommending_you_item_data)



            recommending_you.append(recommending_you_item)


        recommendations_view = cls(
            block_shows=block_shows,
            blocks=blocks,
            codes=codes,
            confirm_page=confirm_page,
            confirm_shows=confirm_shows,
            helpers=helpers,
            items=items,
            lists=lists,
            page_url=page_url,
            profile=profile,
            recommending_you=recommending_you,
        )

        return recommendations_view

