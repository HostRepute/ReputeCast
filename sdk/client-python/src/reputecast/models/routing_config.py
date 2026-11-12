from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.routing_issue import RoutingIssue
  from ..models.routing_named import RoutingNamed
  from ..models.routing_provider import RoutingProvider
  from ..models.routing_shared import RoutingShared
  from ..models.routing_stream_view import RoutingStreamView





T = TypeVar("T", bound="RoutingConfig")



@_attrs_define
class RoutingConfig:
    """ 
        Attributes:
            categories (list[RoutingNamed]):
            issues (list[RoutingIssue]):
            lists (list[RoutingNamed]):
            providers (list[RoutingProvider]):
            saved_by (str):
            sender_domains (list[str]):
            shared (RoutingShared):
            streams (list[RoutingStreamView]):
            templates (list[RoutingNamed]):
            saved_at (datetime.datetime | Unset):
     """

    categories: list[RoutingNamed]
    issues: list[RoutingIssue]
    lists: list[RoutingNamed]
    providers: list[RoutingProvider]
    saved_by: str
    sender_domains: list[str]
    shared: RoutingShared
    streams: list[RoutingStreamView]
    templates: list[RoutingNamed]
    saved_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_issue import RoutingIssue # noqa: PLC0415
        from ..models.routing_named import RoutingNamed # noqa: PLC0415
        from ..models.routing_provider import RoutingProvider # noqa: PLC0415
        from ..models.routing_shared import RoutingShared # noqa: PLC0415
        from ..models.routing_stream_view import RoutingStreamView # noqa: PLC0415
        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.to_dict()
            categories.append(categories_item)



        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        saved_by = self.saved_by

        sender_domains = self.sender_domains



        shared = self.shared.to_dict()

        streams = []
        for streams_item_data in self.streams:
            streams_item = streams_item_data.to_dict()
            streams.append(streams_item)



        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        saved_at: str | Unset = UNSET
        if not isinstance(self.saved_at, Unset):
            saved_at = self.saved_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "issues": issues,
            "lists": lists,
            "providers": providers,
            "savedBy": saved_by,
            "senderDomains": sender_domains,
            "shared": shared,
            "streams": streams,
            "templates": templates,
        })
        if saved_at is not UNSET:
            field_dict["savedAt"] = saved_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_issue import RoutingIssue # noqa: PLC0415
        from ..models.routing_named import RoutingNamed # noqa: PLC0415
        from ..models.routing_provider import RoutingProvider # noqa: PLC0415
        from ..models.routing_shared import RoutingShared # noqa: PLC0415
        from ..models.routing_stream_view import RoutingStreamView # noqa: PLC0415
        d = dict(src_dict)
        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = RoutingNamed.from_dict(categories_item_data)



            categories.append(categories_item)


        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = RoutingIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = RoutingNamed.from_dict(lists_item_data)



            lists.append(lists_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = RoutingProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        saved_by = d.pop("savedBy")

        sender_domains = cast(list[str], d.pop("senderDomains"))


        shared = RoutingShared.from_dict(d.pop("shared"))




        streams = []
        _streams = d.pop("streams")
        for streams_item_data in (_streams):
            streams_item = RoutingStreamView.from_dict(streams_item_data)



            streams.append(streams_item)


        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = RoutingNamed.from_dict(templates_item_data)



            templates.append(templates_item)


        _saved_at = d.pop("savedAt", UNSET)
        saved_at: datetime.datetime | Unset
        if isinstance(_saved_at,  Unset):
            saved_at = UNSET
        else:
            saved_at = datetime.datetime.fromisoformat(_saved_at)




        routing_config = cls(
            categories=categories,
            issues=issues,
            lists=lists,
            providers=providers,
            saved_by=saved_by,
            sender_domains=sender_domains,
            shared=shared,
            streams=streams,
            templates=templates,
            saved_at=saved_at,
        )

        return routing_config

