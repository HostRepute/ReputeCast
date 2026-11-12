from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sender_defaults_input_reply_handling import SenderDefaultsInputReplyHandling
from ..types import UNSET, Unset






T = TypeVar("T", bound="SenderDefaultsInput")



@_attrs_define
class SenderDefaultsInput:
    """ 
        Attributes:
            footer_address (str):
            reply_handling (SenderDefaultsInputReplyHandling):
            default_sender_id (str | Unset): Default From for new campaigns (the Broadcasts default)
            list_unsubscribe_mailto (bool | Unset):
            reply_to (str | Unset):
            tracking_domain (str | Unset): Empty for the provider default
            transactional_sender_id (str | Unset):
            utm_links (bool | Unset):
     """

    footer_address: str
    reply_handling: SenderDefaultsInputReplyHandling
    default_sender_id: str | Unset = UNSET
    list_unsubscribe_mailto: bool | Unset = UNSET
    reply_to: str | Unset = UNSET
    tracking_domain: str | Unset = UNSET
    transactional_sender_id: str | Unset = UNSET
    utm_links: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        footer_address = self.footer_address

        reply_handling = self.reply_handling.value

        default_sender_id = self.default_sender_id

        list_unsubscribe_mailto = self.list_unsubscribe_mailto

        reply_to = self.reply_to

        tracking_domain = self.tracking_domain

        transactional_sender_id = self.transactional_sender_id

        utm_links = self.utm_links


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "footerAddress": footer_address,
            "replyHandling": reply_handling,
        })
        if default_sender_id is not UNSET:
            field_dict["defaultSenderId"] = default_sender_id
        if list_unsubscribe_mailto is not UNSET:
            field_dict["listUnsubscribeMailto"] = list_unsubscribe_mailto
        if reply_to is not UNSET:
            field_dict["replyTo"] = reply_to
        if tracking_domain is not UNSET:
            field_dict["trackingDomain"] = tracking_domain
        if transactional_sender_id is not UNSET:
            field_dict["transactionalSenderId"] = transactional_sender_id
        if utm_links is not UNSET:
            field_dict["utmLinks"] = utm_links

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        footer_address = d.pop("footerAddress")

        reply_handling = SenderDefaultsInputReplyHandling(d.pop("replyHandling"))




        default_sender_id = d.pop("defaultSenderId", UNSET)

        list_unsubscribe_mailto = d.pop("listUnsubscribeMailto", UNSET)

        reply_to = d.pop("replyTo", UNSET)

        tracking_domain = d.pop("trackingDomain", UNSET)

        transactional_sender_id = d.pop("transactionalSenderId", UNSET)

        utm_links = d.pop("utmLinks", UNSET)

        sender_defaults_input = cls(
            footer_address=footer_address,
            reply_handling=reply_handling,
            default_sender_id=default_sender_id,
            list_unsubscribe_mailto=list_unsubscribe_mailto,
            reply_to=reply_to,
            tracking_domain=tracking_domain,
            transactional_sender_id=transactional_sender_id,
            utm_links=utm_links,
        )

        return sender_defaults_input

