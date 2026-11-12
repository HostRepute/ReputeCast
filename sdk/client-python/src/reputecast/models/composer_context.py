from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.composer_audience_option import ComposerAudienceOption
  from ..models.composer_route import ComposerRoute
  from ..models.composer_sender_option import ComposerSenderOption
  from ..models.composer_test_recipient import ComposerTestRecipient
  from ..models.personalization_token import PersonalizationToken





T = TypeVar("T", bound="ComposerContext")



@_attrs_define
class ComposerContext:
    """ 
        Attributes:
            broadcast_route (ComposerRoute):
            lists (list[ComposerAudienceOption]):
            members (list[ComposerTestRecipient]):
            postal_address (str):
            public_url (str):
            segments (list[ComposerAudienceOption]):
            sender_domains (list[str]):
            senders (list[ComposerSenderOption]):
            tokens (list[PersonalizationToken]):
            default_sender_id (str | Unset):
     """

    broadcast_route: ComposerRoute
    lists: list[ComposerAudienceOption]
    members: list[ComposerTestRecipient]
    postal_address: str
    public_url: str
    segments: list[ComposerAudienceOption]
    sender_domains: list[str]
    senders: list[ComposerSenderOption]
    tokens: list[PersonalizationToken]
    default_sender_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_audience_option import ComposerAudienceOption # noqa: PLC0415
        from ..models.composer_route import ComposerRoute # noqa: PLC0415
        from ..models.composer_sender_option import ComposerSenderOption # noqa: PLC0415
        from ..models.composer_test_recipient import ComposerTestRecipient # noqa: PLC0415
        from ..models.personalization_token import PersonalizationToken # noqa: PLC0415
        broadcast_route = self.broadcast_route.to_dict()

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        postal_address = self.postal_address

        public_url = self.public_url

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        sender_domains = self.sender_domains



        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        tokens = []
        for tokens_item_data in self.tokens:
            tokens_item = tokens_item_data.to_dict()
            tokens.append(tokens_item)



        default_sender_id = self.default_sender_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "broadcastRoute": broadcast_route,
            "lists": lists,
            "members": members,
            "postalAddress": postal_address,
            "publicUrl": public_url,
            "segments": segments,
            "senderDomains": sender_domains,
            "senders": senders,
            "tokens": tokens,
        })
        if default_sender_id is not UNSET:
            field_dict["defaultSenderId"] = default_sender_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_audience_option import ComposerAudienceOption # noqa: PLC0415
        from ..models.composer_route import ComposerRoute # noqa: PLC0415
        from ..models.composer_sender_option import ComposerSenderOption # noqa: PLC0415
        from ..models.composer_test_recipient import ComposerTestRecipient # noqa: PLC0415
        from ..models.personalization_token import PersonalizationToken # noqa: PLC0415
        d = dict(src_dict)
        broadcast_route = ComposerRoute.from_dict(d.pop("broadcastRoute"))




        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ComposerAudienceOption.from_dict(lists_item_data)



            lists.append(lists_item)


        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = ComposerTestRecipient.from_dict(members_item_data)



            members.append(members_item)


        postal_address = d.pop("postalAddress")

        public_url = d.pop("publicUrl")

        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = ComposerAudienceOption.from_dict(segments_item_data)



            segments.append(segments_item)


        sender_domains = cast(list[str], d.pop("senderDomains"))


        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = ComposerSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        tokens = []
        _tokens = d.pop("tokens")
        for tokens_item_data in (_tokens):
            tokens_item = PersonalizationToken.from_dict(tokens_item_data)



            tokens.append(tokens_item)


        default_sender_id = d.pop("defaultSenderId", UNSET)

        composer_context = cls(
            broadcast_route=broadcast_route,
            lists=lists,
            members=members,
            postal_address=postal_address,
            public_url=public_url,
            segments=segments,
            sender_domains=sender_domains,
            senders=senders,
            tokens=tokens,
            default_sender_id=default_sender_id,
        )

        return composer_context

