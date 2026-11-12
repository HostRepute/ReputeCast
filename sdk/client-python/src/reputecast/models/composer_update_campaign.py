from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.composer_audience_input import ComposerAudienceInput
  from ..models.composer_content_settings import ComposerContentSettings





T = TypeVar("T", bound="ComposerUpdateCampaign")



@_attrs_define
class ComposerUpdateCampaign:
    """ 
        Attributes:
            audience (ComposerAudienceInput | Unset):
            name (str | Unset):
            reply_to (str | Unset):
            sender_id (str | Unset): A sender id, or empty to clear
            settings (ComposerContentSettings | Unset):
     """

    audience: ComposerAudienceInput | Unset = UNSET
    name: str | Unset = UNSET
    reply_to: str | Unset = UNSET
    sender_id: str | Unset = UNSET
    settings: ComposerContentSettings | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_audience_input import ComposerAudienceInput # noqa: PLC0415
        from ..models.composer_content_settings import ComposerContentSettings # noqa: PLC0415
        audience: dict[str, Any] | Unset = UNSET
        if not isinstance(self.audience, Unset):
            audience = self.audience.to_dict()

        name = self.name

        reply_to = self.reply_to

        sender_id = self.sender_id

        settings: dict[str, Any] | Unset = UNSET
        if not isinstance(self.settings, Unset):
            settings = self.settings.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if audience is not UNSET:
            field_dict["audience"] = audience
        if name is not UNSET:
            field_dict["name"] = name
        if reply_to is not UNSET:
            field_dict["replyTo"] = reply_to
        if sender_id is not UNSET:
            field_dict["senderId"] = sender_id
        if settings is not UNSET:
            field_dict["settings"] = settings

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_audience_input import ComposerAudienceInput # noqa: PLC0415
        from ..models.composer_content_settings import ComposerContentSettings # noqa: PLC0415
        d = dict(src_dict)
        _audience = d.pop("audience", UNSET)
        audience: ComposerAudienceInput | Unset
        if isinstance(_audience,  Unset):
            audience = UNSET
        else:
            audience = ComposerAudienceInput.from_dict(_audience)




        name = d.pop("name", UNSET)

        reply_to = d.pop("replyTo", UNSET)

        sender_id = d.pop("senderId", UNSET)

        _settings = d.pop("settings", UNSET)
        settings: ComposerContentSettings | Unset
        if isinstance(_settings,  Unset):
            settings = UNSET
        else:
            settings = ComposerContentSettings.from_dict(_settings)




        composer_update_campaign = cls(
            audience=audience,
            name=name,
            reply_to=reply_to,
            sender_id=sender_id,
            settings=settings,
        )

        return composer_update_campaign

