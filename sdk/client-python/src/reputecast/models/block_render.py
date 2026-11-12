from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.block_hidden_rec import BlockHiddenRec
  from ..models.block_option import BlockOption
  from ..models.recommendations_block import RecommendationsBlock





T = TypeVar("T", bound="BlockRender")



@_attrs_define
class BlockRender:
    """ 
        Attributes:
            hidden (list[BlockHiddenRec]):
            html (str):
            options (list[BlockOption]):
            page_url (str):
            settings (RecommendationsBlock):
            shows (int):
     """

    hidden: list[BlockHiddenRec]
    html: str
    options: list[BlockOption]
    page_url: str
    settings: RecommendationsBlock
    shows: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.block_hidden_rec import BlockHiddenRec # noqa: PLC0415
        from ..models.block_option import BlockOption # noqa: PLC0415
        from ..models.recommendations_block import RecommendationsBlock # noqa: PLC0415
        hidden = []
        for hidden_item_data in self.hidden:
            hidden_item = hidden_item_data.to_dict()
            hidden.append(hidden_item)



        html = self.html

        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)



        page_url = self.page_url

        settings = self.settings.to_dict()

        shows = self.shows


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hidden": hidden,
            "html": html,
            "options": options,
            "pageUrl": page_url,
            "settings": settings,
            "shows": shows,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.block_hidden_rec import BlockHiddenRec # noqa: PLC0415
        from ..models.block_option import BlockOption # noqa: PLC0415
        from ..models.recommendations_block import RecommendationsBlock # noqa: PLC0415
        d = dict(src_dict)
        hidden = []
        _hidden = d.pop("hidden")
        for hidden_item_data in (_hidden):
            hidden_item = BlockHiddenRec.from_dict(hidden_item_data)



            hidden.append(hidden_item)


        html = d.pop("html")

        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = BlockOption.from_dict(options_item_data)



            options.append(options_item)


        page_url = d.pop("pageUrl")

        settings = RecommendationsBlock.from_dict(d.pop("settings"))




        shows = d.pop("shows")

        block_render = cls(
            hidden=hidden,
            html=html,
            options=options,
            page_url=page_url,
            settings=settings,
            shows=shows,
        )

        return block_render

