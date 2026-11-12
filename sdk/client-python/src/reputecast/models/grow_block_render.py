from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.grow_block_render_hidden import GrowBlockRenderHidden
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_block import ReferralBlock
  from ..models.reward_block import RewardBlock





T = TypeVar("T", bound="GrowBlockRender")



@_attrs_define
class GrowBlockRender:
    """ 
        Attributes:
            count (int):
            for_ (str):
            hidden (GrowBlockRenderHidden):
            html (str):
            list_name (str):
            off_by (str):
            program_id (str):
            reward (RewardBlock):
            settings (ReferralBlock):
            off_at (datetime.datetime | Unset):
     """

    count: int
    for_: str
    hidden: GrowBlockRenderHidden
    html: str
    list_name: str
    off_by: str
    program_id: str
    reward: RewardBlock
    settings: ReferralBlock
    off_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_block import ReferralBlock # noqa: PLC0415
        from ..models.reward_block import RewardBlock # noqa: PLC0415
        count = self.count

        for_ = self.for_

        hidden = self.hidden.value

        html = self.html

        list_name = self.list_name

        off_by = self.off_by

        program_id = self.program_id

        reward = self.reward.to_dict()

        settings = self.settings.to_dict()

        off_at: str | Unset = UNSET
        if not isinstance(self.off_at, Unset):
            off_at = self.off_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "for": for_,
            "hidden": hidden,
            "html": html,
            "listName": list_name,
            "offBy": off_by,
            "programId": program_id,
            "reward": reward,
            "settings": settings,
        })
        if off_at is not UNSET:
            field_dict["offAt"] = off_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_block import ReferralBlock # noqa: PLC0415
        from ..models.reward_block import RewardBlock # noqa: PLC0415
        d = dict(src_dict)
        count = d.pop("count")

        for_ = d.pop("for")

        hidden = GrowBlockRenderHidden(d.pop("hidden"))




        html = d.pop("html")

        list_name = d.pop("listName")

        off_by = d.pop("offBy")

        program_id = d.pop("programId")

        reward = RewardBlock.from_dict(d.pop("reward"))




        settings = ReferralBlock.from_dict(d.pop("settings"))




        _off_at = d.pop("offAt", UNSET)
        off_at: datetime.datetime | Unset
        if isinstance(_off_at,  Unset):
            off_at = UNSET
        else:
            off_at = datetime.datetime.fromisoformat(_off_at)




        grow_block_render = cls(
            count=count,
            for_=for_,
            hidden=hidden,
            html=html,
            list_name=list_name,
            off_by=off_by,
            program_id=program_id,
            reward=reward,
            settings=settings,
            off_at=off_at,
        )

        return grow_block_render

